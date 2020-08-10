pwd = $(shell pwd)

vol1 = $(pwd)/vol1

clean:
	rm -rf $(pwd)/vol*

build: clean
	docker build -t smallcb .

start: clean
	mkdir -p $(vol1)
	docker run -p 8091-8094:8091-8094 \
                   -p 11210:11210 \
                   -v $(vol1):/opt/couchbase/var \
                   --cap-add=SYS_PTRACE \
                   -d smallcb
