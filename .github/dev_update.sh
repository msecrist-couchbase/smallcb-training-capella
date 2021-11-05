#!/bin/bash -x
BUILD_IMAGE="$1"
ARTIFACT_NAME="smallcb-dev.tar.gz"

echo "Updating the environment....`date`"

cd smallcb
sudo make stop-play-server
cd ..
mv smallcb smallcb_`date +%m%d%y_%H%M`
aws s3 cp s3://smallcb-builds/$ARTIFACT_NAME .
mkdir smallcb
tar -xvf $ARTIFACT_NAME -C smallcb
cd smallcb
mkdir rotated
if [ "${BUILD_IMAGE}" = "build_image" ]; then
  sudo make build create
  sudo docker images
fi
make play-server
IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 )
IFS='.' read -r -a array <<< "$IP"
SUBDOMAIN="cb-${array[2]}${array[3]}.couchbase.live"
sudo ./play-server -host ${SUBDOMAIN}  -containers=10 -sessionsMaxAge=35m0s -codeDuration=3m -containersSingleUse=2 -restarters=5 -containerWaitDuration=1m &> nohup.out &

echo "DONE! `date`"