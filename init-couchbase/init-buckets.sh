#!/bin/bash

# expand variables and print commands
set -o xtrace

# exit immediately if a command fails or if there are unset vars
set -euo pipefail

CB_USER="${CB_USER:-Administrator}"
CB_PSWD="${CB_PSWD:-password}"

CB_BUCKET_RAMSIZE="${CB_BUCKET_RAMSIZE:-128}"

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

# Sometimes failing to load the sample without sleep
sleep 10

echo "cbimport beer-sample..."
/opt/couchbase/bin/cbimport json --format sample --verbose \
 -c localhost -u ${CB_USER} -p ${CB_PSWD} \
 -b beer-sample \
 -d file:///opt/couchbase/samples/beer-sample.zip

echo "drop beer-sample indexes..."
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX beer_primary ON `beer-sample`'

echo "create beer-sample primary index..."
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=CREATE PRIMARY INDEX beer_primary ON `beer-sample`'

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

# Sometimes failing to load the sample without sleep
sleep 10

echo "couchbase-cli bucket-list..."
/opt/couchbase/bin/couchbase-cli bucket-list \
 -c localhost -u ${CB_USER} -p ${CB_PSWD}

echo "cbimport travel-sample..."
/opt/couchbase/bin/cbimport json --format sample --verbose \
 -c localhost -u ${CB_USER} -p ${CB_PSWD} \
 -b travel-sample \
 -d file:///opt/couchbase/samples/travel-sample.zip

echo "drop travel-sample indexes..."
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_airportname ON `travel-sample`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_city ON `travel-sample`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_faa ON `travel-sample`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_icao ON `travel-sample`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_airline_primary ON `travel-sample`.`inventory`.`airline`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_airport_airportname ON `travel-sample`.`inventory`.`airport`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_airport_city ON `travel-sample`.`inventory`.`airport`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_airport_primary ON `travel-sample`.`inventory`.`airport`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_airport_faa ON `travel-sample`.`inventory`.`airport`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_hotel_city ON `travel-sample`.`inventory`.`hotel`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_hotel_primary ON `travel-sample`.`inventory`.`hotel`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_landmark_city ON `travel-sample`.`inventory`.`landmark`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_landmark_primary ON `travel-sample`.`inventory`.`landmark`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_route_primary ON `travel-sample`.`inventory`.`route`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_route_route_src_dst_day ON `travel-sample`.`inventory`.`route`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_route_schedule_utc ON `travel-sample`.`inventory`.`route`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_inventory_route_sourceairport ON `travel-sample`.`inventory`.`route`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_name_type ON `travel-sample`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_primary ON `travel-sample`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_route_src_dst_day ON `travel-sample`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_schedule_utc ON `travel-sample`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_sourceairport ON `travel-sample`'
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=DROP INDEX def_type ON `travel-sample`'

# FTS search index
curl -XPUT  http://${CB_USER}:${CB_PSWD}@localhost:8094/api/index/travel-fts-index \
  -H 'cache-control: no-cache' \
  -H 'content-type: application/json' \
  -d '{
 "name": "travel-fts-index",
 "type": "fulltext-index",
 "params": {
  "mapping": {
   "default_mapping": {
    "enabled": true,
    "dynamic": true
   },
   "default_type": "_default",
   "default_analyzer": "standard",
   "default_datetime_parser": "dateTimeOptional",
   "default_field": "_all",
   "store_dynamic": false,
   "index_dynamic": true,
   "docvalues_dynamic": false
  },
  "store": {
   "indexType": "scorch",
   "kvStoreName": ""
  },
  "doc_config": {
   "mode": "type_field",
   "type_field": "type",
   "docid_prefix_delim": "",
   "docid_regexp": ""
  }
 },
 "sourceType": "couchbase",
 "sourceName": "travel-sample",
 "sourceUUID": "",
 "sourceParams": {},
 "planParams": {
  "maxPartitionsPerPIndex": 1,
  "numReplicas": 0,
  "indexPartitions": 1
 },
 "uuid": ""
}'

echo "sleep 10 to allow stabilization..."
sleep 10

echo "create travel-sample primary index..."
curl http://${CB_USER}:${CB_PSWD}@localhost:8093/query/service \
    -d 'statement=CREATE PRIMARY INDEX def_primary ON `travel-sample`'

echo "couchbase-cli bucket-list..."
/opt/couchbase/bin/couchbase-cli bucket-list \
 -c localhost -u ${CB_USER} -p ${CB_PSWD}

echo "couchbase-cli bucket-edit beer-sample..."
/opt/couchbase/bin/couchbase-cli bucket-edit \
 -c localhost -u ${CB_USER} -p ${CB_PSWD} \
 --bucket beer-sample \
 --bucket-replica 0

echo "couchbase-cli bucket-edit travel-sample..."
/opt/couchbase/bin/couchbase-cli bucket-edit \
 -c localhost -u ${CB_USER} -p ${CB_PSWD} \
 --bucket travel-sample \
 --bucket-replica 0

echo "couchbase-cli bucket-list..."
/opt/couchbase/bin/couchbase-cli bucket-list \
 -c localhost -u ${CB_USER} -p ${CB_PSWD}

echo "couchbase-cli rebalance..."
/opt/couchbase/bin/couchbase-cli rebalance \
 -c localhost -u ${CB_USER} -p ${CB_PSWD} --no-progress-bar

echo "sleep 40 to allow stabilization..."
sleep 40
