#!/bin/bash

# exit immediately if a command fails or if there are unset vars
set -euo pipefail

# turn on bash's job control, used to bring couchbase-server back to
# the forground after the node is configured
set -m

# prepend script to /etc/service/couchbase-server/run script...
cp /init-couchbase/init-service-run.txt \
   /etc/service/couchbase-server/run.tmp

cat /etc/service/couchbase-server/run | tail -n +3 >> \
    /etc/service/couchbase-server/run.tmp

mv /etc/service/couchbase-server/run.tmp \
   /etc/service/couchbase-server/run

chmod +x /etc/service/couchbase-server/run

# append to /opt/couchbase/etc/couchbase/static_config...
cat /init-couchbase/init-static-config.txt >> \
    /opt/couchbase/etc/couchbase/static_config

echo "Starting couchbase-server..."
/entrypoint.sh couchbase-server &

sleep 5

echo "Restarting couchbase-server..."
/opt/couchbase/bin/couchbase-server -k

sleep 5

echo "Waiting for couchbase-server..."
until curl -s http://localhost:8091/pools > /dev/null; do
    sleep 5
    echo "Waiting for couchbase-server..."
done

echo "Waiting for couchbase-server... ready"

if ! couchbase-cli server-list -c localhost:8091 -u Administrator -p password > /dev/null; then
  echo "couchbase cluster-init..."
  couchbase-cli cluster-init \
        --services data,query,index,fts \
        --index-storage-setting default \
        --cluster-ramsize 1024 \
        --cluster-index-ramsize 256 \
        --cluster-fts-ramsize 256 \
        --cluster-eventing-ramsize 0 \
        --cluster-analytics-ramsize 0 \
        --cluster-username Administrator \
        --cluster-password password \
        --cluster-name smallcb
fi

sleep 3

echo "Reconfiguring indexer..."
curl -v -X POST -d @/init-couchbase/init-indexer.json \
     http://Administrator:password@127.0.0.1:9102/internal/settings?internal=ok

sleep 3

killall indexer

sleep 3

curl http://Administrator:password@127.0.0.1:9102/internal/settings?internal=ok | jq .

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

sleep 5

