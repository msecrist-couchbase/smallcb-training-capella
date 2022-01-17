#!/bin/bash -x
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1
exec > /tmp/couchbase-live.log 2>&1

su - ubuntu

#Configure golang for the root user
export GOROOT=/usr/local/go
export GOPATH=/home/ubuntu/go
export GOBIN=$GOPATH/bin
export GOCACHE="/home/ubuntu/.cache/go-build"
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
source ~/.profile

# pull docker image
docker pull 598307997273.dkr.ecr.us-west-1.amazonaws.com/smallcb-beta:latest
docker tag 598307997273.dkr.ecr.us-west-1.amazonaws.com/smallcb-beta:latest smallcb:latest

cd /home/ubuntu

# download smallcb artifact
sudo -u ubuntu aws s3 cp s3://smallcb-builds/smallcb-staging.tar.gz .

# Create working directory
sudo -u ubuntu mkdir smallcb

# Extract smallcb
sudo -u ubuntu tar -xvf smallcb-staging.tar.gz -C smallcb

# CD to the working dir
cd smallcb

# Create nginx config
chmod a+x ./devops/add_nginx_ssl_listeners.sh
sudo sh -c './devops/add_nginx_ssl_listeners.sh > /etc/nginx/sites-available/playground'
sudo ln -s /etc/nginx/sites-available/playground /etc/nginx/sites-enabled/playground
sudo rm /etc/nginx/sites-enabled/default
sudo systemctl restart nginx


#Create the directory where the rotated logs will be stored
sudo -u ubuntu mkdir rotated

IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 )
IFS='.' read -r -a array <<< "$IP"
SUBDOMAIN="cb-${array[2]}${array[3]}.couchbase.live"
echo "Subdomain of this node = $SUBDOMAIN"

# Added below for resolving cannot kill Docker container - permission denied
sudo systemctl disable apparmor.service --now

#Start SmallCb
./play-server -host "$SUBDOMAIN" -egressHandlerUrl="http://internal-smallcb-capella-egress-beta-1883733566.us-west-1.elb.amazonaws.com/" -containers=10 -sessionsMaxAge=35m0s -codeDuration=3m -containersSingleUse=2 -restarters=5 -containerWaitDuration=3m -tlsTerminalProxy  &> nohup.out &

