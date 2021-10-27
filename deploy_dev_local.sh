#!/bin/bash
############################################
# Helper script for quick deployment to dev
#
############################################

ACTION="$1"
BRANCH="$2"
BUILD_IMAGE="$3"
echo "Dev playground environment..."

: ${ACTION:="help"}

help()
{
  echo "Usage: $0 checkout|artifacts|deploy|all [branch] [build_image]"
  exit 1
}

checkout()
{
  if [ "$BRANCH" == "" ];then
    echo "WARNING: BRANCH is not specified"
    help
    exit 1
  fi
  if [ -d smallcb ]; then
    mv smallcb smallcb_`date +%m%d%y_%H%M`
  fi
  git clone -b ${BRANCH} http://github.com/couchbaselabs/smallcb.git
  cd smallcb
  make build create play-server
}

artifacts()
{
  tar -cvzf smallcb-dev.tar.gz *
  aws s3 cp smallcb-dev.tar.gz s3://smallcb-builds/
  aws s3 ls s3://smallcb-builds/
}

deploy()
{
  aws ec2 describe-instances --filters Name=tag-value,Values=smallcb-dev --query 'Reservations[*].Instances[*].[InstanceId,PublicIpAddress,State.Name]' --output text |egrep running
INSTANCE_ID="`aws ec2 describe-instances --filters Name=tag-value,Values=smallcb-dev --query 'Reservations[*].Instances[*].[InstanceId,State.Name]' --output text |egrep running |xargs -n 2 | cut -f1 -d' '`"
 echo mssh -o StrictHostKeyChecking=no ubuntu@${INSTANCE_ID} -C /home/ubuntu/dev_update.sh ${BUILD_IMAGE}
 mssh -o StrictHostKeyChecking=no ubuntu@${INSTANCE_ID} -C /home/ubuntu/dev_update.sh ${BUILD_IMAGE}
}

all()
{
  checkout
  artifacts
  deploy
}

$ACTION

echo "Done"
