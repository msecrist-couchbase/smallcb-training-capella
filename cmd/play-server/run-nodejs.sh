#!/bin/bash

export PATH=$PATH:/home/play/npm-packages/bin

export NODE_PATH=/home/play/npm-packages/lib/node_modules

export NPM_PACKAGES=/home/play/npm-packages

cd $(dirname ${1})

node code.nodejs
