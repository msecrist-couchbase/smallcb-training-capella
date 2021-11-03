#!/bin/bash
source /etc/profile.d/rvm.sh

cd $(dirname ${1})

COUCHBASE_BACKEND_LOG_LEVEL=error ruby -W0 code.rb
