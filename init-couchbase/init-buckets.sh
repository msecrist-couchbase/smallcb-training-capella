#!/bin/bash

CB_USER="${CB_USER:-Administrator}"
CB_PSWD="${CB_PSWD:-password}"

CB_BUCKET_RAMSIZE="${CB_BUCKET_RAMSIZE:-128}"

# exit immediately if a command fails or if there are unset vars
set -euo pipefail

echo "couchbase-cli bucket-create beer-sample..."
/opt/couchbase/bin/couchbase-cli bucket-create \
 -c localhost -u ${CB_USER} -p ${CB_PSWD} \
 --bucket beer-sample \
 --bucket-type couchbase \
 --bucket-ramsize ${CB_BUCKET_RAMSIZE} \
 --bucket-replica 0 \
 --bucket-priority low \
 --bucket-eviction-policy fullEviction \
 --enable-flush 1 \
 --enable-index-replica 0 \
 --wait

echo "cbdocloader beer-sample..."
/opt/couchbase/bin/cbdocloader \
 -c localhost -u ${CB_USER} -p ${CB_PSWD} \
 -b beer-sample \
 -m ${CB_BUCKET_RAMSIZE} \
 -v \
 -d /opt/couchbase/samples/beer-sample.zip

echo "couchbase-cli bucket-create travel-sample..."
/opt/couchbase/bin/couchbase-cli bucket-create \
 -c localhost -u ${CB_USER} -p ${CB_PSWD} \
 --bucket travel-sample \
 --bucket-type couchbase \
 --bucket-ramsize ${CB_BUCKET_RAMSIZE} \
 --bucket-replica 0 \
 --bucket-priority low \
 --bucket-eviction-policy fullEviction \
 --enable-flush 1 \
 --enable-index-replica 0 \
 --wait

echo "cbdocloader travel-sample..."
/opt/couchbase/bin/cbdocloader \
 -c localhost -u ${CB_USER} -p ${CB_PSWD} \
 -b travel-sample \
 -m ${CB_BUCKET_RAMSIZE} \
 -v \
 -d /opt/couchbase/samples/travel-sample.zip

# echo "couchbase bucket-create test..."
# couchbase-cli bucket-create \
#         --cluster localhost \
#         --username ${CB_USER} \
#         --password ${CB_PSWD} \
#         --bucket test \
#         --bucket-type couchbase \
#         --bucket-ramsize ${CB_BUCKET_RAMSIZE} \
#         --wait

