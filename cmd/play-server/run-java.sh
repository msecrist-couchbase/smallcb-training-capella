#!/bin/bash

export JAVA_HOME=/opt/java/openjdk

export PATH=$PATH:/opt/java/openjdk/bin

cd $(dirname ${1})

javac -cp .:`cat /home/play/hello/classpath.txt` code.java

java -cp .:`cat /home/play/hello/classpath.txt` Program
