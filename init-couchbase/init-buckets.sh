#!/bin/bash

# exit immediately if a command fails or if there are unset vars
set -euo pipefail

echo "cbdocloader beer-sample..."
/opt/couchbase/bin/cbdocloader \
        -c localhost -u Administrator -p password \
        -b beer-sample -m 128 -v \
	-d /opt/couchbase/samples/beer-sample.zip

# echo "cbdocloader travel-sample..."
# /opt/couchbase/bin/cbdocloader \
#         -c localhost -u Administrator -p password \
#         -b travel-sample -m 128 -v \
# 	-d /opt/couchbase/samples/travel-sample.zip

# echo "couchbase bucket-create test..."
# couchbase-cli bucket-create \
#         --cluster localhost \
#         --username Administrator \
#         --password password \
#         --bucket test \
#         --bucket-type couchbase \
#         --bucket-ramsize 128 \
#         --wait

