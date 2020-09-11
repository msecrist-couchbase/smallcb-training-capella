#!/bin/bash

export GOPATH=/go

export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

mkdir -p $(dirname ${1})/.gocache

export GOCACHE=$(dirname ${1})/.gocache

cd $(dirname ${1})

go run code.go
