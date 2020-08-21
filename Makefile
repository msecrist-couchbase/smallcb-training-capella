# IMAGE_NAME is the docker image name.
IMAGE_NAME = smallcb

# CONTAINER_NAME is the docker container name.
CONTAINER_NAME = smallcb1

# Build the docker image.
build:
	rm -rf vol-data*
	docker build -t $(IMAGE_NAME) .

# Start a docker container instance, init it, and stop it (but keep it
# around -- don't delete it), in order to create the vol-data-snapshot
# subdirectory.
create:
	rm -rf vol-data*
	mkdir -p vol-data/
	docker run -p 8091-8094:8091-8094 -p 11210:11210 \
                   -v vol-data:/opt/couchbase/var \
                   --cap-add=SYS_PTRACE \
                   --name=$(CONTAINER_NAME) \
                   -d $(IMAGE_NAME)
	sleep 3
	docker exec $(CONTAINER_NAME) /init-couchbase/init.sh
	sleep 3
	docker stop $(CONTAINER_NAME)
	sleep 3
	cp -R vol-data/ vol-data-snapshot/

# Restart the docker container instance from the vol-data-snapshot.
restart:
	docker stop $(CONTAINER_NAME) || true
	rm -rf vol-data/*
	cp -R vol-data-snapshot/ vol-data/
	docker start $(CONTAINER_NAME)

# After a restart, wait until couchbase-server is healthy via polling.
restart-wait: restart
	echo "Checking couchbase-server healthy..."
	until \
           curl http://Administrator:password@127.0.0.1:8091/pools/default/buckets | jq . | grep healthy; \
        do \
           sleep 1; \
        done
