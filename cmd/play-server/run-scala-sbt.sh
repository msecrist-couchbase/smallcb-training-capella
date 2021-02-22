#!/bin/bash
FILEPATH=${1}
if [ "${FILEPATH}" != "" ]; then
    if [ -d ${FILEPATH} ]; then
        cd ${FILEPATH}
    else
        cd $(dirname ${FILEPATH})
    fi
fi
cp /run-scala-build.sbt ./build.sbt
sbt run
