#!/bin/bash

cd $(dirname ${1})

IS_SEC_RESTRICT=`egrep -e "import[ .]+[']?os|platform|pkg_resources|socket|urllib|base64|server|internal[']?[ .]*" code.py`
if [ "${IS_SEC_RESTRICT}" != "" ]; then
    echo "Security restriction: some packages are not allowed."
    exit 1
fi
python3 code.py
