FROM couchbase

RUN apt-get update && \
    apt-get install -y \
	git curl wget \
	atop htop psmisc \
	emacs

COPY init-sample-buckets.sh /init-sample-buckets.sh
RUN chmod +x /init-sample-buckets.sh
RUN /init-sample-buckets.sh
