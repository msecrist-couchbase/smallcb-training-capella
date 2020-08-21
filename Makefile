# IMAGE_NAME is the docker image name.
IMAGE_NAME = smallcb

# CONTAINER_NAME is the docker container name.
CONTAINER_NAME = smallcb1

# Build the docker image.
build:
	rm -rf data*
	docker build -t $(IMAGE_NAME) .

# Start a docker container instance, init it, and stop it (but keep it
# around -- don't delete it), in order to create the data-snapshot
# subdirectory.
create:
	rm -rf data*
	mkdir -p data/
	docker run -p 8091-8094:8091-8094 -p 11210:11210 \
                   -v data:/opt/couchbase/var \
                   --cap-add=SYS_PTRACE \
                   --name=$(CONTAINER_NAME) \
                   -d $(IMAGE_NAME)
	sleep 3
	docker exec $(CONTAINER_NAME) /init-couchbase/init.sh
	sleep 3
	docker stop $(CONTAINER_NAME)
	sleep 3
	cp -R data/ data-snapshot/

# Restart the docker container instance from the data-snapshot.
restart:
	docker stop $(CONTAINER_NAME) || true
	rm -rf data/*
	cp -R data-snapshot/ data/
	docker start $(CONTAINER_NAME)
