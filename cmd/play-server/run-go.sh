#!/bin/bash

export GOPATH=/go

export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

cd $(dirname ${1})/..

go run ./code
