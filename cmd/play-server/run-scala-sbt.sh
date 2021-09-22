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
sbt --batch -Dsbt.server.forcestart=true run
