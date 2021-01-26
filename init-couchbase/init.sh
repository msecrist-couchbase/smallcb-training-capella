#!/bin/bash

CB_USER="${CB_USER:-Administrator}"
CB_PSWD="${CB_PSWD:-password}"
CB_HOST="${CB_HOST:-127.0.0.1}"
CB_PORT="${CB_PORT:-8091}"
CB_NAME="${CB_NAME:-play}"

CB_SERVICES="${CB_SERVICES:-data,query,index,fts}"

CB_KV_RAMSIZE="${CB_KV_RAMSIZE:-1024}"
CB_INDEX_RAMSIZE="${CB_INDEX_RAMSIZE:-256}"
CB_FTS_RAMSIZE="${CB_FTS_RAMSIZE:-256}"
CB_EVENTING_RAMSIZE="${CB_EVENTING_RAMSIZE:-0}"
CB_ANALYTICS_RAMSIZE="${CB_ANALYTICS_RAMSIZE:-0}"

CB_INDEXER_PORT="${CB_INDEXER_PORT:-9102}"

# exit immediately if a command fails or if there are unset vars
set -euo pipefail

# turn on bash's job control, used to bring couchbase-server
# back to the foreground after the node is configured
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
until curl -s http://${CB_HOST}:${CB_PORT}/pools > /dev/null; do
    sleep 5
    echo "Waiting for couchbase-server..."
done

echo "Waiting for couchbase-server... ready"

if ! couchbase-cli server-list -c ${CB_HOST}:${CB_PORT} -u ${CB_USER} -p ${CB_PSWD} > /dev/null; then
  echo "couchbase cluster-init..."
  couchbase-cli cluster-init \
        --services ${CB_SERVICES} \
        --cluster-name ${CB_NAME} \
        --cluster-username ${CB_USER} \
        --cluster-password ${CB_PSWD} \
        --cluster-ramsize ${CB_KV_RAMSIZE} \
        --cluster-index-ramsize ${CB_INDEX_RAMSIZE} \
        --cluster-fts-ramsize ${CB_FTS_RAMSIZE} \
        --cluster-eventing-ramsize ${CB_EVENTING_RAMSIZE} \
        --cluster-analytics-ramsize ${CB_ANALYTICS_RAMSIZE}
fi

sleep 3

echo "Reconfiguring indexer..."
curl -v -X POST -d @/init-couchbase/init-indexer.json \
     http://${CB_USER}:${CB_PSWD}@${CB_HOST}:${CB_INDEXER_PORT}/internal/settings?internal=ok

sleep 3

killall indexer

sleep 3

curl http://${CB_USER}:${CB_PSWD}@${CB_HOST}:${CB_INDEXER_PORT}/internal/settings?internal=ok | jq .
