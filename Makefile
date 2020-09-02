IMAGE_NAME = smallcb

CONTAINER_NUM = 0

CONTAINER_PORTS = -p 8091-8096:8091-8096 -p 11210:11210

CONTAINER_EXTRAS = --cap-add=SYS_PTRACE

# -------------------------------------------------

# Build the docker image.
build:
	rm -rf vol-*
	docker build -t $(IMAGE_NAME) .

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
	docker run --name=$(IMAGE_NAME)-$(CONTAINER_NUM) \
                   $(CONTAINER_PORTS) $(CONTAINER_EXTRAS) \
                   -v $(shell pwd)/vol-snapshot:/opt/couchbase/var \
                   -d $(IMAGE_NAME)
	sleep 3
	docker exec $(IMAGE_NAME)-$(CONTAINER_NUM) /init-couchbase/init.sh
	sleep 3
	docker exec $(IMAGE_NAME)-$(CONTAINER_NUM) /init-couchbase/init-buckets.sh
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
	cp -R vol-snapshot/ vol-instances/vol-$(CONTAINER_NUM)/
	docker run --name=$(IMAGE_NAME)-$(CONTAINER_NUM) \
                   $(CONTAINER_PORTS) $(CONTAINER_EXTRAS) \
                   -v $(shell pwd)/vol-instances/vol-$(CONTAINER_NUM):/opt/couchbase/var \
                   -d $(IMAGE_NAME)

# -------------------------------------------------

wait-healthy:
	echo "Waiting until couchbase-server is healthy..."
	time docker exec $(IMAGE_NAME)-$(CONTAINER_NUM) /init-couchbase/wait-healthy.sh

# -------------------------------------------------

play-server-src = \
        cmd/play-server/main.go \
        cmd/play-server/main_flags.go \
        cmd/play-server/main_template.go \
        cmd/play-server/misc.go \
        cmd/play-server/run.go \
        cmd/play-server/session.go

play-server: $(play-server-src)
	go build ./...
