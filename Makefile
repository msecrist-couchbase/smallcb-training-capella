IMAGE_NAME = smallcb

IMAGE_FROM = couchbase

CONTAINER_NUM = 0

CONTAINER_PORTS = -p 8091-8096:8091-8096 -p 11210:11210

# To enable strace diagnosis, use...
# CONTAINER_EXTRAS = --cap-add=SYS_PTRACE
CONTAINER_EXTRAS =

SERVICE_HOST = couchbase.live

CB_ADMIN_PASSWORD = small-house-secret

# -------------------------------------------------

# Build the docker image.
build:
	rm -rf vol-*
	cat Dockerfile-before \
            Dockerfile-from-$(IMAGE_FROM) \
            Dockerfile-include-sdk \
            Dockerfile-suffix > \
            Dockerfile
	docker build -t $(IMAGE_NAME) .
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
	mkdir -p vol-snapshot
	mkdir -p tmp
	docker run --name=$(IMAGE_NAME)-$(CONTAINER_NUM) \
               $(CONTAINER_PORTS) $(CONTAINER_EXTRAS) \
               -v $(shell pwd)/vol-snapshot/:/opt/couchbase/var \
               -d $(IMAGE_NAME)
	sleep 3
	docker exec $(IMAGE_NAME)-$(CONTAINER_NUM) /init-couchbase/init.sh
	sleep 3
	docker exec $(IMAGE_NAME)-$(CONTAINER_NUM) /init-couchbase/init-buckets.sh
	sleep 3
	docker exec $(IMAGE_NAME)-$(CONTAINER_NUM) /opt/couchbase/bin/couchbase-cli \
               reset-admin-password --new-password $(CB_ADMIN_PASSWORD)
	sleep 3
	docker cp $(IMAGE_NAME)-$(CONTAINER_NUM):/opt/couchbase/VERSION.txt ./tmp/VERSION.txt
	docker exec $(IMAGE_NAME)-$(CONTAINER_NUM) \
               grep vsn /opt/couchbase/lib/ns_server/erlang/lib/ns_server/ebin/ns_server.app | \
               cut -d '"' -f 2 > ./tmp/ns_server.app.vsn
	sleep 3
	docker stop $(IMAGE_NAME)-$(CONTAINER_NUM)
	sleep 3
	docker rm $(IMAGE_NAME)-$(CONTAINER_NUM)
	sleep 3
	rm -rf vol-snapshot/lib/couchbase/logs/*
	rm -rf vol-snapshot/lib/couchbase/stats/*

# -------------------------------------------------

# Restart the docker container instance and wait until its
# couchbase-server is healthy.
restart: restart-snapshot wait-healthy

# Restart the docker container instance from the vol-snapshot.
restart-snapshot:
	docker stop $(IMAGE_NAME)-$(CONTAINER_NUM) || true
	docker rm $(IMAGE_NAME)-$(CONTAINER_NUM) || true
	rm -rf vol-instances/vol-$(CONTAINER_NUM)/*
	mkdir -p vol-instances/vol-$(CONTAINER_NUM)
	cp -R vol-snapshot/* vol-instances/vol-$(CONTAINER_NUM)/
	docker run --name=$(IMAGE_NAME)-$(CONTAINER_NUM) \
               $(CONTAINER_PORTS) $(CONTAINER_EXTRAS) \
               -v $(shell pwd)/vol-instances/vol-$(CONTAINER_NUM)/:/opt/couchbase/var \
               --add-host="$(SERVICE_HOST):127.0.0.1" \
               -d $(IMAGE_NAME)

# -------------------------------------------------

wait-healthy:
	echo "Waiting until couchbase-server is healthy..."
	time docker exec -e CB_PSWD=$(CB_ADMIN_PASSWORD) \
                    $(IMAGE_NAME)-$(CONTAINER_NUM) /init-couchbase/wait-healthy.sh

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

play-server: $(play-server-src)
	go build ./...
