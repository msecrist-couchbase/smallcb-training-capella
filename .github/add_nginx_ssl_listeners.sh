#!/bin/bash
HOST="$1"
if [ "${HOST}" == "" ]; then
  echo "Usage: $0 HOST"
  echo "Example: $0 cb-9380.couchbase.live"
  exit 1
fi

echo 'server {
   server_name '"${HOST}"';

   location / {
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Host $http_host;
        proxy_pass http://'"${HOST}"':8080;
    }

    listen 443 ssl;
    listen [::]:443 ssl;
    ssl_certificate /var/www/'"${HOST}"'/cert/playground.crt;
    ssl_certificate_key /var/www/'"${HOST}"'/cert/playground.key;
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
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Host $http_host;
        proxy_pass http://'"${HOST}"':'"${PORT}"';
    }
    listen '"${SSL}"' ssl;
    listen [::]:'"${SSL}"' ssl;
    ssl_certificate /var/www/'"${HOST}"'/cert/playground.crt;
    ssl_certificate_key /var/www/'"${HOST}"'/cert/playground.key;
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
        proxy_pass http://'"${HOST}"':'"${PORT}"';
    }
    listen '"${SSL}"' ssl;
    listen [::]:'"${SSL}"' ssl;
    ssl_certificate /var/www/'"${HOST}"'/cert/playground.crt;
    ssl_certificate_key /var/www/'"${HOST}"'/cert/playground.key;
    access_log   /var/log/nginx/'"${HOST}"'.access.log;
    error_log    /var/log/nginx/'"${HOST}"'.error.log;
}'
 done
done

