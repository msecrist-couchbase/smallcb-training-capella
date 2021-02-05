#!/bin/bash

# exit immediately if a command fails or if there are unset vars
set -euo pipefail

if [ -d /etc/service/ ]; then
   # prepend script to /etc/service/couchbase-server/run script...
   cp /init-couchbase/prep-service-run.txt \
      /etc/service/couchbase-server/run.tmp

   cat /etc/service/couchbase-server/run | tail -n +3 >> \
      /etc/service/couchbase-server/run.tmp

   mv /etc/service/couchbase-server/run.tmp \
      /etc/service/couchbase-server/run

   chmod +x /etc/service/couchbase-server/run
fi