#!/bin/bash

cd $(dirname ${1})

scalac -classpath .:`cat /home/play/helloscala/scala-classpath.txt` code.scala

scala -classpath .:`cat /home/play/helloscala/scala-classpath.txt` -J-Dlog4j.configuration=file:/run-scala-log4j.properties com.couchbase.Program