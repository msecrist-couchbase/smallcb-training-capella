#!/bin/bash

# exit immediately if a command fails, or a command in a pipeline
# fails, or if there are unset variables
set -euo pipefail

# turn on bash's job control, used to bring couchbase-server back to
# the forground after the node is configured
set -m

echo "Starting couchbase-server..."
/entrypoint.sh couchbase-server &

until curl -s http://localhost:8091/pools > /dev/null; do
    sleep 5
    echo "Checking couchbase-server..."
done

echo "Checking couchbase-server... ready"

# check if cluster is already initialized
# if ! couchbase-cli server-list -c localhost:8091 -u Administrator -p password > /dev/null; then
# fi
#
# fg 1

echo "couchbase cluster-init..."
couchbase-cli cluster-init \
        --services data,index,query \
        --index-storage-setting default \
        --cluster-ramsize 1024 \
        --cluster-index-ramsize 256 \
        --cluster-analytics-ramsize 0 \
        --cluster-eventing-ramsize 0 \
        --cluster-fts-ramsize 0 \
        --cluster-username Administrator \
        --cluster-password password \
        --cluster-name tkick

echo "couchbase bucket-create test..."
couchbase-cli bucket-create \
        --cluster localhost \
        --username Administrator \
        --password password \
        --bucket test \
        --bucket-type couchbase \
        --bucket-ramsize 128 \
        --wait

echo "cbdocloader beer-sample..."
/opt/couchbase/bin/cbdocloader \
        -c localhost -u Administrator -p password \
        -b beer-sample -m 128 -v \
	-d /opt/couchbase/samples/beer-sample.zip

echo "cbdocloader travel-sample..."
/opt/couchbase/bin/cbdocloader \
        -c localhost -u Administrator -p password \
        -b travel-sample -m 128 -v \
	-d /opt/couchbase/samples/travel-sample.zip

sleep 5

