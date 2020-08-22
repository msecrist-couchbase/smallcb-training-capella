ARG CB_EDITION=enterprise
ARG CB_VERSION=6.6.0
ARG CB_IMAGE=couchbase:$CB_EDITION-$CB_VERSION

FROM $CB_IMAGE

RUN apt-get update && \
    apt-get install -y \
	git curl wget jq \
	build-essential cmake \
	atop htop psmisc strace \
	emacs golang-go

# Install python SDK, see:
# https://docs.couchbase.com/tutorials/quick-start/quickstart-python3-native-firstquery-cb65.html

RUN apt-get install -y \
    python3-dev python3-pip python3-setuptools && \
    pip3 install couchbase

# Copy init-couchbase files into image.

RUN mkdir -p /init-couchbase

COPY init-couchbase /init-couchbase

RUN chmod +x /init-couchbase/*.sh

