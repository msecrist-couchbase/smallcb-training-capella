#!/bin/bash
export JAVA_HOME=/opt/java/openjdk
export PATH=$PATH:/opt/java/openjdk/bin

FILEPATH=${1}
if [ "${FILEPATH}" != "" ]; then
    if [ -d ${FILEPATH} ]; then
        cd ${FILEPATH}
    else
        cd $(dirname ${FILEPATH})
    fi
fi
cp /run-scala-build.sbt ./build.sbt
cp /run-scala-log4j.properties ./log4j.properties
sbt --error --batch -Dsbt.server.forcestart=true -J-Dlog4j.configuration=file:./log4j.properties run
