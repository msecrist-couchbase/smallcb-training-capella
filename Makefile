# IMAGE_NAME is the docker image name.
IMAGE_NAME = smallcb

# CONTAINER_NAME is the docker container instance name.
CONTAINER_NAME = smallcb1

pwd = $(shell pwd)

vol1 = $(pwd)/vol1

clean:
	rm -rf $(pwd)/vol*

# Build the image.
build: clean
	docker build -t $(IMAGE_NAME) .

# Start and init the named container instance.
start: clean
	mkdir -p $(vol1)
	docker run -p 8091-8094:8091-8094 -p 11210:11210 \
                   -v $(vol1):/opt/couchbase/var \
                   --cap-add=SYS_PTRACE \
                   --name=$(CONTAINER_NAME) \
                   -d $(IMAGE_NAME)
	sleep 3
	docker exec $(CONTAINER_NAME) /init-couchbase/init.sh
