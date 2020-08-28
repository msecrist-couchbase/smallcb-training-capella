IMAGE_NAME = smallcb

CONTAINER_NUM = 0

PORTS = -p 8091-8094:8091-8094 -p 11210:11210

# Build the docker image.
build:
	rm -rf vol-*
	docker build -t $(IMAGE_NAME) .

# Create the vol-snapshot directory.
#
# This is done by starting a container instance, initializing the
# couchbase server with sample data -- configured for lower resource
# utilization footprint -- and then stopping/removing the container
# instance, while keeping the vol-snapshot directory.
create:
	rm -rf vol-*
	mkdir -p vol-snapshot
	docker run $(PORTS) \
                   -v $(shell pwd)/vol-snapshot:/opt/couchbase/var \
                   --cap-add=SYS_PTRACE \
                   --name=$(IMAGE_NAME)-$(CONTAINER_NUM) \
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

# Restart the docker container instance and wait until its
# couchbase-server is healthy.
restart: restart-snapshot wait-healthy

# Restart the docker container instance from the vol-snapshot.
restart-snapshot:
	docker stop $(IMAGE_NAME)-$(CONTAINER_NUM) || true
	docker rm $(IMAGE_NAME)-$(CONTAINER_NUM) || true
	rm -rf vol-$(CONTAINER_NUM)/*
	cp -R vol-snapshot/ vol-$(CONTAINER_NUM)/
	docker run $(PORTS) \
                   -v $(shell pwd)/vol-$(CONTAINER_NUM):/opt/couchbase/var \
                   --cap-add=SYS_PTRACE \
                   --name=$(IMAGE_NAME)-$(CONTAINER_NUM) \
                   -d $(IMAGE_NAME)

wait-healthy:
	echo "Waiting until couchbase-server is healthy..."
	time docker exec $(IMAGE_NAME)-$(CONTAINER_NUM) /init-couchbase/wait-healthy.sh

play-server: cmd/play-server/main.go
	go build ./...
