IMAGE_NAME = smallcb

CONTAINER_NUM = 0

# Build the docker image.
build:
	rm -rf vol-*
	docker build -t $(IMAGE_NAME) .

# Start a docker container instance, init it, and remove it, with the
# goal of creating and capturing the vol-snapshot subdirectory.
create:
	rm -rf vol-*
	mkdir -p vol-snapshot
	docker run -p 8091-8094:8091-8094 -p 11210:11210 \
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

# Restart the docker container instance from the vol-snapshot.
restart-snapshot:
	docker stop $(IMAGE_NAME)-$(CONTAINER_NUM) || true
	docker rm $(IMAGE_NAME)-$(CONTAINER_NUM) || true
	rm -rf vol-$(CONTAINER_NUM)/*
	cp -R vol-snapshot/ vol-$(CONTAINER_NUM)/
	docker run -p 8091-8094:8091-8094 -p 11210:11210 \
                   -v $(shell pwd)/vol-$(CONTAINER_NUM):/opt/couchbase/var \
                   --cap-add=SYS_PTRACE \
                   --name=$(IMAGE_NAME)-$(CONTAINER_NUM) \
                   -d $(IMAGE_NAME)

# After restart-snapshot, wait until couchbase-server is healthy.
restart: restart-snapshot
	echo "Checking couchbase-server healthy..."
	until \
           curl http://Administrator:password@127.0.0.1:8091/pools/default/buckets | jq . | grep healthy; \
        do \
           sleep 1; \
        done

play-server: cmd/play-server/main.go
	go build ./...
