#!/bin/bash

until curl http://Administrator:password@127.0.0.1:8091/pools/default/buckets | jq . | grep healthy; do \
    sleep 1;
done
