#!/bin/bash

PUBIP=$(curl http://169.254.169.254/latest/meta-data/public-ipv4)
HOST="cb-$(cut -d. -f3<<<$PUBIP)$(cut -d. -f4<<<$PUBIP).couchbase.live"

echo 'server {
   server_name '"${HOST}"';

   location / {
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Host $http_host;
        proxy_pass http://localhost:8080;
    }

    listen 443 ssl;
    listen [::]:443 ssl;
    ssl_certificate /etc/nginx/playground.crt;
    ssl_certificate_key /etc/nginx/playground.key;
    access_log   /var/log/nginx/'"${HOST}"'.access.log;
    error_log    /var/log/nginx/'"${HOST}"'.error.log;
}

server {
    if ($host = '"${HOST}"') {
        return 301 https://$host$request_uri;
    }

   listen 80;
   listen [::]:80;

   server_name    '"${HOST}"';
    return 404;
}'

# direct ports
for PORT in `echo 8091`
do
   SSL="`expr 20000 + ${PORT}`"
   echo 'server {
   server_name '"${HOST}"';

   location / {
        proxy_pass http://localhost:'"${PORT}"';
    }
    listen '"${SSL}"' ssl;
    listen [::]:'"${SSL}"' ssl;
    ssl_certificate /etc/nginx/playground.crt;
    ssl_certificate_key /etc/nginx/playground.key;
    access_log   /var/log/nginx/'"${HOST}"'.access.log;
    error_log    /var/log/nginx/'"${HOST}"'.error.log;
}'
done

# container proxies
for C in `seq 0 9`
do
 for B in `echo 1 2 3 4 5 6 40 41 42`
 do
	 PORT="`expr 10000 + 100 \* $C + $B`"
	 SSL="`expr 20000 + 100 \* $C + $B`"
   echo 'server {
   server_name '"${HOST}"';
    
   location / {
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Host $http_host;
        proxy_pass http://localhost:'"${PORT}"';
    }
    listen '"${SSL}"' ssl;
    listen [::]:'"${SSL}"' ssl;
    ssl_certificate /etc/nginx/playground.crt;
    ssl_certificate_key /etc/nginx/playground.key;
    access_log   /var/log/nginx/'"${HOST}"'.access.log;
    error_log    /var/log/nginx/'"${HOST}"'.error.log;
}'
 done
done
