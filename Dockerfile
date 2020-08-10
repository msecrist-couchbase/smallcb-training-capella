FROM couchbase

RUN apt-get update && \
    apt-get install -y \
	git curl wget jq \
	atop htop psmisc strace \
	emacs golang-go

RUN mkdir -p /init-couchbase

COPY init-couchbase /init-couchbase

RUN chmod +x /init-couchbase/*.sh
