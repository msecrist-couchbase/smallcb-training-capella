#!/bin/bash

CB_USER="${CB_USER:-Administrator}"
CB_PSWD="${CB_PSWD:-password}"
CB_HOST="${CB_HOST:-127.0.0.1}"
CB_PORT="${CB_PORT:-8091}"

until curl http://${CB_USER}:${CB_PSWD}@${CB_HOST}:${CB_PORT}/pools/default/buckets | jq . | grep healthy; do \
    sleep 1;
done
