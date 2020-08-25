#!/bin/bash

cd $(dirname ${1})

javac $(basename ${1})

java Program
