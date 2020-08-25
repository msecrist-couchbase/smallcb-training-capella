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

RUN apt-get install -y python3-dev python3-pip python3-setuptools && \
    pip3 install couchbase

# TODO: Need couchbase java SDK, but seems like have to drag in maven?

RUN apt-get install -y openjdk-8-jdk

# TODO: Install nodejs problem -- need a non-root user so npm install -g works?

# Install nodejs SDK, see:
# https://docs.couchbase.com/tutorials/getting-started-ce/dev-nodejs/tutorial_en.html
# https://github.com/nodesource/distributions/blob/master/README.md

# RUN curl -sL https://deb.nodesource.com/setup_14.x | bash - && \
#     apt-get install -y nodejs npm && \
#     npm install -g couchbase ottoman

# Copy init-couchbase files into image.

RUN mkdir -p /init-couchbase
COPY init-couchbase /init-couchbase
RUN chmod +x /init-couchbase/*.sh

# Copy play-server related files into image.

COPY cmd/play-server/run-java.sh /run-java.sh
RUN chmod +x /run-java.sh
