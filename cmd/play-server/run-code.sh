#!/bin/bash

mkdir -p /opt/couchbase/var/tmp/play

chown -R play:couchbase /opt/couchbase/var/tmp/play

su -c "${2} ${3}" ${1}
