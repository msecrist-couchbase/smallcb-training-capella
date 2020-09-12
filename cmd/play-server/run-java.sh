#!/bin/bash

export JAVA_HOME=/opt/java/openjdk

export PATH=$PATH:/opt/java/openjdk/bin

cd $(dirname ${1})

javac code.java

java Program
