IMAGE_NAME ?= smallcb

IMAGE_FROM ?= couchbase

CONTAINER_NAME ?= smallcb

CONTAINER_NUM ?= 0

CONTAINER_PORTS ?= -p 8091-8096:8091-8096 -p 11210:11210

# To enable strace diagnostics during development, use...
# CONTAINER_EXTRAS = --cap-add=SYS_PTRACE
CONTAINER_EXTRAS ?=

CREATE_EXTRAS ?=

BUILD_EXTRAS ?=

SERVICE_HOST ?= couchbase.live

SNAPSHOT_SUFFIX ?=

CB_ADMIN_PASSWORD ?= small-house-secret

CLITOOLS_IMAGE_NAME ?= smallcb-clitools

GOOS?=

GOARCH?=

# -------------------------------------------------

# Build the docker image.
# Note: duplicated for CI in .github/workflows/staging.yaml
build:
	rm -rf vol-*
	cat Dockerfile-args \
            Dockerfile-from-$(IMAGE_FROM) \
            Dockerfile-include-sdk \
            Dockerfile-suffix > \
            Dockerfile
	docker build --network host -t $(IMAGE_NAME) ${BUILD_EXTRAS} .
	rm Dockerfile

build-clitools:
	cat Dockerfile-args \
            Dockerfile-from-$(IMAGE_FROM) \
			Dockerfile-clitools > \
            Dockerfile
	docker build --network host -t $(CLITOOLS_IMAGE_NAME) ${BUILD_EXTRAS} .
	rm Dockerfile

# Create the vol-snapshot directory.
#
# This is done by starting a container instance, initializing the
# couchbase server with sample data -- configured for lower resource
# utilization (at the potential tradeoff of performance, etc) -- and
# then stopping/removing the container instance, while keeping the
# vol-snapshot directory (for later reuse/restart'ing).
create:
	rm -rf vol-*
	mkdir -p vol-snapshot$(SNAPSHOT_SUFFIX)
	mkdir -p tmp
	docker run --name=$(CONTAINER_NAME)-$(CONTAINER_NUM) \
               $(CONTAINER_PORTS) $(CONTAINER_EXTRAS) \
               -v $(shell pwd)/vol-snapshot$(SNAPSHOT_SUFFIX)/:/opt/couchbase/var \
               -d $(IMAGE_NAME)
	sleep 3
	docker exec -u root $(CONTAINER_NAME)-$(CONTAINER_NUM) bash -c "${CREATE_EXTRAS} /init-couchbase/init.sh"
	sleep 3
	docker exec -u root $(CONTAINER_NAME)-$(CONTAINER_NUM) /init-couchbase/init-buckets.sh
	sleep 3
	docker exec -u root $(CONTAINER_NAME)-$(CONTAINER_NUM) /opt/couchbase/bin/couchbase-cli \
               reset-admin-password --new-password $(CB_ADMIN_PASSWORD)
	sleep 3
	docker cp $(CONTAINER_NAME)-$(CONTAINER_NUM):/opt/couchbase/VERSION.txt ./tmp/VERSION.txt
	docker exec -u root $(CONTAINER_NAME)-$(CONTAINER_NUM) \
               grep vsn /opt/couchbase/lib/ns_server/erlang/lib/ns_server/ebin/ns_server.app | \
               cut -d '"' -f 2 > ./tmp/ns_server.app.vsn
	for f in `docker exec $(CONTAINER_NAME)-$(CONTAINER_NUM) /bin/sh -c 'ls /opt/couchbase/VERSION-sdk*.ver'`; \
               do docker cp $(CONTAINER_NAME)-$(CONTAINER_NUM):$${f} ./tmp/; done
	sleep 3
	docker stop $(CONTAINER_NAME)-$(CONTAINER_NUM)
	sleep 3
	docker rm $(CONTAINER_NAME)-$(CONTAINER_NUM)
	sleep 3
	ls -la vol-snapshot$(SNAPSHOT_SUFFIX)/lib/couchbase/logs
	whoami
	rm -rf vol-snapshot$(SNAPSHOT_SUFFIX)/lib/couchbase/logs/* || echo "Failed to cleanup logs"
	rm -rf vol-snapshot$(SNAPSHOT_SUFFIX)/lib/couchbase/stats/* || echo "Failed to cleanup stats"
	cp -R cmd/play-server/static vol-snapshot$(SNAPSHOT_SUFFIX)/ || echo "Failed to copy static files"

# -------------------------------------------------

# Restart the docker container instance and wait until its
# couchbase-server is healthy.
restart: pull-image restart-snapshot wait-healthy

# Pull smallcb image (only if its remote)
pull-image:
	echo $(IMAGE_NAME) | grep '/' && docker pull $(IMAGE_NAME) || echo ignoring-err-docker-pull

# Restart the docker container instance from the vol-snapshot.
restart-snapshot: instance-stop snapshot-reset instance-start

# -------------------------------------------------

instance-stop:
	docker stop $(CONTAINER_NAME)-$(CONTAINER_NUM) || echo ignoring-err-docker-stop
	docker rm --force $(CONTAINER_NAME)-$(CONTAINER_NUM) || echo ignoring-err-docker-rm

instance-start:
	docker run --rm --name=$(CONTAINER_NAME)-$(CONTAINER_NUM) \
               $(CONTAINER_PORTS) $(CONTAINER_EXTRAS) \
               -v $(shell pwd)/vol-instances/vol-$(CONTAINER_NUM)/:/opt/couchbase/var \
               --add-host="$(SERVICE_HOST):127.0.0.1" \
               -d $(IMAGE_NAME)

instance-pause:
	docker pause $(CONTAINER_NAME)-$(CONTAINER_NUM)

instance-unpause:
	docker unpause $(CONTAINER_NAME)-$(CONTAINER_NUM) || echo ignoring-err-docker-unpause

# -------------------------------------------------

snapshot-reset:
	rm -rf vol-instances/vol-$(CONTAINER_NUM)/*
	mkdir -p vol-instances/vol-$(CONTAINER_NUM)
	cp -R vol-snapshot$(SNAPSHOT_SUFFIX)/* vol-instances/vol-$(CONTAINER_NUM)/

# -------------------------------------------------

wait-healthy:
	echo "Waiting until couchbase-server is healthy..."
	time docker exec -e CB_PSWD=$(CB_ADMIN_PASSWORD) \
                    $(CONTAINER_NAME)-$(CONTAINER_NUM) /init-couchbase/wait-healthy.sh

# -------------------------------------------------

play-server-src = \
        cmd/play-server/admin.go \
        cmd/play-server/cookie.go \
        cmd/play-server/http.go \
        cmd/play-server/http_proxy.go \
        cmd/play-server/main.go \
        cmd/play-server/main_flags.go \
        cmd/play-server/main_template.go \
        cmd/play-server/misc.go \
        cmd/play-server/restarter.go \
        cmd/play-server/run.go \
        cmd/play-server/run_session.go \
        cmd/play-server/session.go \
        cmd/play-server/captcha.go

play-server: test-play-server build-play-server

build-play-server: $(play-server-src)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build ./cmd/play-server

test-play-server: $(play-server-src)
	go test ./cmd/play-server

start-play-server:
	./play-server -sessionsMaxAge=60m -sessionsMaxIdle=60m -containers=2 -restarters=2 -codeDuration=3m -containerWaitDuration=1m &
	sleep 10

stop-play-server:
	pkill play-server || true
	docker rm $(shell docker ps -aq) --force || true

test-examples: start-play-server
	pip3 install -r tests/requirements.txt
	CBLIVE_URL=http://localhost:8080 CODE_DIR=cmd/play-server/static/examples \
		python3 tests/cblive_playground_runcodetest.py


image: 
	docker build . -f Dockerfile-playserver -t play-server

# Running the gen-examples program depends on other projects,
# which need to be checked out as sibling directories
# to this smallcb directory...
#
# - github.com/couchbaselabs/sdk-examples
# - github.com/couchbase/docs-sdk-dotnet
# - github.com/couchbase/docs-sdk-java
# - github.com/couchbase/docs-sdk-nodejs
# - github.com/couchbase/docs-sdk-php
# - github.com/couchbase/docs-sdk-python
# - github.com/couchbase/docs-server
#
gen-examples-run:
	go fmt ./cmd/gen-examples
	rm -f ./gen-examples
	go build ./cmd/gen-examples
	rm -f ./cmd/play-server/static/examples/gen_*
	./gen-examples

