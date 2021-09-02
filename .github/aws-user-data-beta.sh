#!/bin/bash -x
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1
exec > /tmp/couchbase-live.log 2>&1
sudo apt -y update
sudo apt -u upgrade

su - ubuntu

#Adjust logrotate config to split files on 500k chunks
sed -i 's/1k/500k/' /home/ubuntu/smallcb-logrotate.conf 

#Adjust crontab to run as the ubuntu user and every 30 minutes
sudo sed -i 's/root\tlogrotate/ubuntu\tsudo logrotate/' /etc/crontab
sudo sed -i 's/5  \*/*\/30  \*/' /etc/crontab

#Configure golang for the root user
export GOROOT=/usr/local/go
export GOPATH=/home/ubuntu/go
export GOBIN=$GOPATH/bin
export GOCACHE="/home/ubuntu/.cache/go-build"
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
source ~/.profile

#Without this command, couchbase will fail to restart during make create 
# sudo docker system prune -a -f

# Pull smallcb runner image
export IMAGE_NAME='598307997273.dkr.ecr.us-west-1.amazonaws.com/smallcb-beta:latest'
docker pull "$IMAGE_NAME"

#Forward port 80 to 8080 
sudo iptables -t nat -I PREROUTING -p tcp --dport 80 -j REDIRECT --to-ports 8080
cd /home/ubuntu

# download smallcb artifact
sudo -u ubuntu aws s3 cp s3://smallcb-builds/smallcb-staging.tar.gz .

# Create working directory
sudo -u ubuntu mkdir smallcb

# Extract smallcb
sudo -u ubuntu tar -xvf smallcb-staging.tar.gz -C smallcb

# CD to the working dir
cd smallcb

#Create the directory where the rotated logs will be stored
sudo -u ubuntu mkdir rotated

IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 )
IFS='.' read -r -a array <<< "$IP"
SUBDOMAIN="cb-${array[2]}${array[3]}.couchbase.live"
echo "Subdomain of this node = $SUBDOMAIN"

#Start SmallCb
./play-server -host "$SUBDOMAIN"  -containers=10 -sessionsMaxAge=35m0s -codeDuration=60s -containersSingleUse=2 -restarters=5 &> nohup.out &
