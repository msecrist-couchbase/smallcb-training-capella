IMAGE_NAME = smallcb

CONTAINER_NUM = 0

PORTS = -p 8091-8094:8091-8094 -p 11210:11210

# Build the docker image.
build:
	rm -rf vol-*
	docker build -t $(IMAGE_NAME) .

# Start a docker container instance, init it, and remove it, with the
# goal of creating and capturing the vol-snapshot subdirectory.
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
	docker stop $(IMAGE_NAME)-$(CONTAINER_NUM)
	sleep 3
	docker rm $(IMAGE_NAME)-$(CONTAINER_NUM)
	sleep 3
	rm -rf vol-snapshot/lib/couchbase/logs/*
	rm -rf vol-snapshot/lib/couchbase/stats/*

# Restart the docker container instance and wait until
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
	docker exec $(IMAGE_NAME)-$(CONTAINER_NUM) /init-couchbase/wait-healthy.sh

play-server: cmd/play-server/main.go
	go build ./...
