#!/bin/bash

cd $(dirname ${1})
FILE=$(basename ${1})
cp /run-cc.makefile ./Makefile
sed "s/code/${FILE}/g" /run-cc.cmakelists >./CMakeLists.txt
make
EXEC_FILE=${FILE%%.*}
./${EXEC_FILE}