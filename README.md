README for couchbase playground

* aka, "couchbase.live"
* aka, "small house couchbase" / "smallcb"

-------------------------
Dependencies...

* make
* docker
* golang
  * tip: after checking out this project, run "go get ./..."
    to download golang dependencies.
    * tip: you might need to setup your GOPATH env variables.
      * example: GOPATH=/Users/steve.yen/go

Tip: Denis Rosa offers this as what worked for him
for his golang setup on mac OSX (using brew)...

    export GOPATH="${HOME}/.go"
    export GOROOT="$(brew --prefix golang)/libexec"
    export PATH="$PATH:$GOPATH/bin:$GOROOT/bin"

Tip: Matthew Groves offers these extra steps that
he used to have golang work properly for him...

    go get github.com/go-macaron/captcha
    go get github.com/google/uuid
    go get gopkg.in/yaml.v2

* python 3
-------------------------
Instructions to build and run...

One time setup/init/build steps...

    # 1) To create the docker image 'smallcb',
    # which includes both couchbase-server & couchbase sdks...
    
    make build

    # 2) To create the vol-snapshot directory...
    #
    # TIP: Make sure you're not already running couchbase-server
    # as it needs the standard couchbase-server port #'s.
    
    make create

    # 3) Compile the playground web/app server...
    
    make play-server

And, to start the web/app server...

    ./play-server

The ./play-server command starts a web/app server that listens on port
8080, 8091 and 8093 for web browser requests.  TIP: Make sure you're
not already using those ports, or use the command-line flags to change
the port #'s.
    
For command-line usage/help...

    ./play-server -h

Example usage during development...

    ./play-server -sessionsMaxAge=60m -sessionsMaxIdle=60m -containers=2 -restarters=2 -codeDuration=3m

During development, to see the playground web page, visit...

      http://127.0.0.1:8080

During development, if you modify any of the Dockerfile parts
or the ./init-couchbase parts, you will have to run `make build`
and `make create` again.  If you change the golang code, you'll
have to run `make play-server` again.

Changes to the static files (e.g., HTML templates, CSS, examples, etc)
should not need a re-make and should be immediately visible in
the next web browser page refresh.

To run the tests (all examples on all languages), then execute the below (which would start the playserver and run the tests).
  make test-examples

To start the play server:
  make start-play-server

To stop the play server:
  make stop-play-server


------------------------
TLS/SSL support
 Flags: 
 -tlsTerminalProxy  : To use the https lister ports in the links and is based on default, -tlsListenerBase=20000 
    NOTE: Make sure nginx has beenn started with all these ssl port listeners. A new helper script available at .github/add_nginx_ssl_listeners.sh to generate the config to copy at /etc/nginx/sites-available/playground and make a link at /etc/nginx/sites-enabled/playground.

 -tlsServer : To start the play-server in TLS mode
 -tlsTerminal : To start the gritty in TLS mode
 -tlsKey : Supply the key file path
 -tlsCert : supply the cert file path

Production example:
   nohup ./play-server -host cb-9380.couchbase.live -baseUrl beta.couchbase.live -containers=10 -sessionsMaxAge=35m0s -codeDuration=3m -containersSingleUse=2 -restarters=5 -containerWaitDuration=1m -tlsTerminalProxy &

  To start the playserver itself (if required without front-ending with nginx) with https including gritty process, add the below to the above command.
-tlsTerminal -tlsServer -tlsKey /var/www/cb-9380.couchbase.live/cert/playground.key -tlsCert /var/www/cb-9380.couchbase.live/cert/playground.crt
  NOTE: Delete port fowarding rule 80 to 8080 so that any non-ssl will be forwarded to ssk using nginx automatically 
  Example:
    root@ip-10-0-1-169:/home/ubuntu/smallcb# sudo iptables -t nat -v -L PREROUTING -n --line-number
Chain PREROUTING (policy ACCEPT 263 packets, 14264 bytes)
num   pkts bytes target     prot opt in     out     source               destination         
1     1109 64412 REDIRECT   tcp  --  *      *       0.0.0.0/0            0.0.0.0/0            tcp dpt:80 redir ports 8080
2     1956  113K REDIRECT   tcp  --  *      *       0.0.0.0/0            0.0.0.0/0            tcp dpt:80 redir ports 8080
3    74194 4118K DOCKER     all  --  *      *       0.0.0.0/0            0.0.0.0/0            ADDRTYPE match dst-type LOCAL
root@ip-10-0-1-169:/home/ubuntu/smallcb# sudo iptables -t nat -D PREROUTING 1

------------------------
Dynamic Egress API
Purpose: To allow Capella sessions without wide outbound open (0.0.0.0/0) with dynamic security group creation for the IP addresses of cluster nodes and then associate with Playserver running aws instance id.
API is ELB -> Lambda (python boto3)

New flag:
  -egressHandlerUrl='http://internal-smallcb-capella-egress-beta-1883733566.us-west-1.elb.amazonaws.com/'
  -egressHandlerUrl='' (No dynamic security group and association)

Example for beta env node:
  nohup ./play-server -host cb-151238.couchbase.live -baseUrl beta.couchbase.live -egressHandlerUrl='http://internal-smallcb-capella-egress-beta-1883733566.us-west-1.elb.amazonaws.com/' -containers=10 -sessionsMaxAge=35m0s -codeDuration=3m -containersSingleUse=2 -restarters=5 -containerWaitDuration=3m -tlsTerminalProxy &

Lambda function code 
  cmd/play-server/aws_sg_handler.py
Lambda function permission policy:
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeSecurityGroupRules",
                "ec2:DescribeVpcs",
                "ec2:CreateSecurityGroup",
                "ec2:DeleteSecurityGroup",
                "ec2:ModifySecurityGroupRules",
                "ec2:DescribeSecurityGroups",
                "ec2:AuthorizeSecurityGroupEgress",
                "ec2:RevokeSecurityGroupEgress",
                "ec2:ModifyInstanceAttribute",
                "ec2:DescribeInstances",
                "ec2:CreateTags"
            ],
            "Resource": "*"
        }
    ]
}

-------------------------
Production usage should set the CB_ADMIN_PASSWORD env
variable for security and the host parameter that
represents the publically visible DNS hostname...

    CB_ADMIN_PASSWORD=no-longer-the-small-house-secret \
      ./play-server \
        -host couchbase.live \
        -containers=5 -containersSingleUse=2 -restarters=5

-------------------------
YAML files are used to define more content, including...

- cmd/play-server/static/examples - Code Examples
- cmd/play-server/static/tours - Guided Tours
- cmd/play-server/static/inject.yaml - contextual / API Help

-------------------------
Aside...

    # To create the docker image 'smallcb-sdks',
    # which includes only the couchbase sdks and
    # which does not include couchbase server...
    
    make IMAGE_FROM=base IMAGE_NAME=smallcb-sdks build

-------------------------
TODO's...

how about staging?
  first "real" staging setup on cloud?

need a "down for maintenance" static web page path or toggle?

roughly, how much will it cost?

need a cgroup or a throwaway container to safely run
  submitted code with hard timeouts / resource limits?

lots of examples are auto-scraped from docs and
  put into: https://github.com/couchbaselabs/sdk-examples
    every once in awhile.
  see also: https://github.com/couchbaselabs/devguide-examples
    which are older.
  see the ./cmd/gen-examples program.
    many samples aren't passing gen-example's filters.

call-to-action / download couchbase?

CSS styling for mobile / narrow screens?

better 404 error screen?

client-side JS to prevent over-clicking on Run/Submit button,
  where the current approach has the browser
  correctly handling over-clicking automatically
  w.r.t. requests, but does not have the right UX,
  where the second click doesn't disable the 'run' button
  or make it look different.

client-side JS to prevent Back Button or navigating away
  from losing work in the code textarea?

when there are enough examples, use a tree-control
  on the left-hand-side with a scrollable panel,
  perhaps with mouseovers with longer explanations?
  and/or a tags based filter?

favorites / recommended / starred examples?

feedback comments or votes on examples?

CAPTCHA random seed looks weak
  seeing repeats on process start?

CAPTCHA panics sometimes in its rand.Intn() code?

can i have >1 longer running instance per email?
  ANS: currently, sorta -- full name + email must be unique.

what if my email and/or name are already used?
  if so, can i get another zipcar for me?
  ANS: currently, sorta -- full name + email must be unique.

should we also dump stats to S3 occasionally, not only logs?

keep average time of restarts,
  for fake/estimated progress/ETA bars?
  or, "we are under heavy load" messages?

health watchers and elastic scaling -- bring up more nodes
  when there's more people and auto-scale-down when
  traffic goes away?

sizing?
  some rough disk usage info -- beer-sample + travel-sample

    % du -s -h vol-instances/vol-*
    50M	vol-instances/vol-0
    51M	vol-instances/vol-1

    % du -s -h vol-snapshot
    40M	vol-snapshot

cpu sizing?

inject better UI into web admin UI?

output (stdout / stderr) is not streaming?

docker on mac OSX sometimes gets 'stuck' -- container
  instances aren't restartable sometimes?  Need to sometimes
  restart entire docker system / hypervisor... will this be the
  case also on linux (where we haven't seen this on linux so far)?
  If so, perhaps need a "kill the
  entire server/machine and replace it" button?

pkill -u play might not be using a strong enough kill signal?

should we use docker build env vars?

should we use docker on docker?

use docker networking features?
  use docker network overlay --internal mode?
  perhaps too complex?
  perhaps necessary if we want a play run-only
    container for submitted code?

use tmpfs for faster restarts and less real i/o,
  at the cost of RAM?
  docker run --tmpfs flag?

docker run has interesting tweakable runtime resource limits
  to look at and perhaps use?

docker run --read-only flag?

PHP code examples?

SECURITY: turn off egress networking?
  turn off ability to initiate outbound connections?
  https://www.reddit.com/r/docker/comments/hvs7n9/how_do_i_prevent_a_container_from_making_outgoing/
  If your container is hosted on a VM in Azure, AWS, GCP, OpenStack
  etc, you'll want to restrict Egress (outgoing) traffic
  or new outbound connections from the host
  via Security Rules on the private network?

can admin URL's be put behind an ELB password?

SECURITY: docker build can set ulimits
  and optional security-opts?
  See:
    https://docs.couchbase.com/server/current/install/best-practices-vm.html
    docker run -d --ulimit nofile=40960:40960 \
      --ulimit core=100000000:100000000 \
      --ulimit memlock=100000000:100000000 \
      --name db -p 8091-8096:8091-8096 -p 11210-11211:11210-11211 couchbase

SECURITY: perhaps only allow host to connect (or proxy),
  and listen on 127.0.0.1 instead of 0.0.0.0?

SECURITY: cpu/memory usage limits?

SECURITY: restart the host system every day?
  just in case that unsafe code escapes
  the container sandbox via kernel hacks and such?

SECURITY: bad actor can start a webserver and host porn, etc?

SECURITY: bad actor can DDoS other systems (need egress control)?

configure AWS to limit network bandwidth ingress / egress?

SECURITY: hosting IAM rules?

SECURITY: need email spam/flood prevention throttling?
  e.g., swatting prevention?

SECURITY: need a bad-list of emails that we don't like?

influitive hookups?

copy/pastable connection snippets for popular languages
  and SDK's, for >= zipcar mode when there's a session?

the proxy serving of web-admin UI login screen
  when not logged in always goes to container 0,
  but what if container 0 is restarting?
  Then the UI login screen can't be served?
  Maybe play-server's proxy should cache?

popup tours in UI injection?
  https://kamranahmed.info/driver.js/

/static URL should not have directory listing

multi-request-with-data-freezing/thawing
  (hertz/avid, multi-day rental)
  after a timeout from inactivity,
  the data is snapshotted and parked in quiescent garage somewhere...
    like on to local disk,
      or onto S3.
  when the user comes back, data is thawed,
    against a restarted container,
    perhaps at a different assigned port #'s?
    which takes some time (e.g., go get a coffee) while defrosting?

finally, if you want a lease of 1 or more fleet of cars
  (allowing for clustering),
  with attached pool of on-demand hotel chauffeurs and
    mechanic/maintenance services...
  then use Couchbase Cloud.

-------------------------
On new CB version release...

Does a new CB version mean a new EC2 instance
  and then redirect the DNS / ELB,
  a'la rolling upgrade?
  For now, just run latest version only.

What about frozen data --
  do we thaw them on demand, as requested?
  eventually give up on versions that's too old?

How about GC'ing data that is super old?

What about GDPR with emails and PII?

-------------------------
use cases
  try it now
    open-ended tire kicking?
    of SDK / API testing?
       and/or N1QL?
       and/or query workbench?
       and workload generator?
    of sync-gateway / mobile?
    try-it-now buttons in the docs & tutorials?

more use cases with persistent data?
  CI/CD tests?

  backend jobs on-demand?
    on-demand analytics?
    quick slice/dice jobs against big data (public datasets: covid19)?
    AI/ML jobs?

  serverless event processing?

poor man's sizing estimator / guesstimator?

docker pause/unpause feature to reduce footprint?
  tried this -- see commit 228f20f744126ed49
  and although it works for macbook local dev with localhost
  and on a test EC2 instance with direct IP address access,
  leading to very low CPU usage with the paused container instances,
  the web-admin UI for Couchbase (port 8091) does not seem
  to work in production via the cb-XXYY.couchbase.live domain names,
  so need to figure that out.

dev-mode config is reusable for laptops, too?

dev-mode still asking for stats too much?

dev-mode, phase II, needs core product improvements?

race / raciness in N1QL server where the running of user code
  is faster than N1QL server's ability -- where N1QL
  server sometimes (incorrectly!) returns an empty result
  instead of a not-yet-ready kind of error.  Example curl N1QL
  output with empty results...

  {
  "requestID": "6aabfdf7-7700-4f1c-8c8b-a29ed29f036f",
  "signature": {"*":"*"},
  "results": [
  ],
  "status": "success",
  "metrics": {"elapsedTime": "171.8859ms","executionTime": "171.5529ms","resultCount": 0,"resultSize": 0}
  }
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed

  0     0    0     0    0     0      0      0 --:--:-- --:--:-- --:--:--     0
  100   329  100   222  100   107   1190    573 --:--:-- --:--:-- --:--:--  1193

proxy inject guided tour JS?
  https://introjs.com/
  or equivalents
  or copy-snippet of code

--------------------------
DONE: store logs on S3? - DONE: by Denis.

DONE: figure out where to run this in production? - DONE: by Denis.

DONE: google analytics using same script JS as from developer.couchbase.com

DONE: iframe for run output emits HTML instead of text/plain
  so that we have google analytics for run output.

DONE: max session duration (-sessionsMaxAge) and
  max session idle/inactivity duration (-sessionsMaxIdle).

DONE: some examples only make sense when there's
  a session (>= zipcar mode), so the example's YAML
  should define a property of...

    className: needs-session

  ...so that the example will only appear when
  the user has a session.

DONE: some containers are dedicated to be session-less
  like 10-items-or-less quick checkout lanes...
  see: -containersSingleUse cmd-line param, defaults to 0.

DONE: first end-to-end demo on laptop.

DONE: first end-to-end demo on cloud -- thanks Denis Rosa for ec2 setup!

DONE: UI should show the CB version -- in the footer.

DONE: RBAC to limit access, using generated UUID for username and password.

DONE: CAPTCHA with 5 second sleep on bad guess.

DONE: session URL takes optional example 'e' target param,
  which is used as the target page on successful session start,
  like... http://couchbase.live/session?e=examples/basic-n1ql

DONE: su to play user to install java SDK, nodejs SDK.

DONE: line numbers for code textarea via ACE code editor.

DONE: SECURITY: remove ability to strace in production,
  via default of CONTAINER_EXTRAS in Makefile.
  CVE-2014-4699: A bug in ptrace() could allow privilege
  escalation. Docker disables ptrace() inside the container using
  apparmor, seccomp and by dropping CAP_PTRACE.

DONE: when the page is scrolled down, and you click on another
  example link, there's a disconcerting jump back to
  top of the page rather than having the page stay
  mostly stable -- similar to docs behavior,
  which is now improved by using anchor links.

DONE: UI timeouts for long-running programs, see:
  codeMaxDuration and containerWaitDuration.

DONE: SECURITY: after a run times out in a session,
  pkill all the processes owned by 'play' user.
  Use case: cbworkloadgen or any submitted program
    is thus prevented from running longer than the
    play-server's codeDuration timeout.

DONE: use another user 'play' to run user submitted code...
  docker exec as -u play:couchbase (user:group),
    not as root and not as couchbase.

DONE: examples can now be collected into separate "books" --
  multiple example subdirectories are now supported.

DONE: InfoBefore / InfoAfter can now have HTML markup,
  like links to relevant docs page or "next" step links.

DONE: one-click workload generator, done via cbworkloadgen.

DONE: create couchbase user better than Administrator:password,
  especially dynamically with user & password that look
  more like UUID's when we're in >= zipcar mode.

DONE: change the Administrator password in 'make create'.

DONE: using docker --add-host to add to /etc/hosts file so that
  couchbase://couchbase.live:8091 connection string works.

DONE: need to golang proxy to use the remapped port #'s
  in the REST responses, to rewrite REST json maps
  to list server hostnames/addrs correctly.

DONE: docker container is now -p or publish'ing or exposing ports
  on 0.0.0.0 addr instead of 127.0.0.1 addr by default.
  See also "-listen" cmd-line flag.

DONE: in the header, added link to docs.

DONE: capture full name and email to get a longer-running
  dev tire-kicking instance (or zipcar mode).

DONE: CAPTCHA when starting test-drive session.

DONE: inject play-server UI parts into web admin UI.

DONE: where do we store full name & email -- ANS: log to stdout.

DONE: examine nginx proxy for subdomain based routing -- ANS: not easy.
  For example, the following, but which unfortunately doesn't seem to handle
  multiple virtual ports...
  https://blog.florianlopes.io/host-multiple-websites-on-single-host-docker/

DONE: use docker checkpoint to help speed up slow restart times...
  ANS: no, checkpoints are currently only an experimental feature
       in docker (2020/09/04).

DONE: need ping / sanity checking REST endpoints
  see /static/test.txt
  and /admin/stats
  and /admin/dashboard

DONE: proxy n1ql 8093 port so that command-line copy/paste of curl example works.

DONE: couchbase style header / footer (logo added).

DONE: first internal show & tell.
DONE: first examples from not-steve.

DONE: client-side code syntax highlighting / styling via
      ACE code editor?

DONE: DNS, ELB & subdomains.

DONE: ELB and sticky load balancing.
  ELB / ALB probably won't work due to kv/2i/fts/services ports
  and non-HTTP protocols.
  Might need to use explicit domain names.
    c01.couchbase.live.
    c02.couchbase.live.
    c03.couchbase.live.

DONE: look into HOSTALIASES env var for mapping hosts to ip addresses -- not needed.

DONE: proper web terminal output UI.

DONE: PHP SDK (by jagadesh m).

DONE: iframe for access to web admin portal --
  ANS: nope, because...
    ns-server does not like iframes, so the proxy removes
      the X-Frame-Options DENY header from the response.
    need server-side proxy in golang?
    perhaps access to just query workbench?

DONE: pop up web admin portal in separate tab or browser target.

--------------------------
DONE: handwave design ideas...

start with couchbase docker image...

  --------------------
  tweak & configure couchbase to lower TCO
  init couchbase
  tweak & configure couchbase to lower TCO, part 2
  create bucket(s)
  load sample data

  --------------------
  add language 1 tools
  add sdk 1 V1 stuff
  add sdk 1 V2 stuff

  add language N tools
  add sdk N V1 stuff
  add sdk N V2 stuff

  --------------------
  then, freeze or snapshot /opt/couchbase/var
    as a good restart point

--------------------------
DONE: also, put faster changing stuff into host filesystem
  via docker volumes feature
  for easier github updates?
    without having update the snapshots?

  e.g.,
    per-language samples & sample apps
    tutorials / try-it-now templates
    more sample data

  only works for IaaS/EC2,
    and won't work with ECS/GCS/AzureCS or FarGate,
    as we don't have access to host system?

--------------------------
DONE: it might connect to localhost couchbase,
 or might connect to remotehost couchbase,
   for longer persistence?

DONE: on the host
  web/app-server
    which starts container
      (e.g., docker run -rm SAFE SAFE whatever),
    communicates via stdin/stdout,
      perhaps using "docker attach CONTAINER"?
    when done, shuts down the container instance
      and restarts it cleanly (ahead of time),
        in preparation for next request,
        to reduce cold start window?

DONE: have a pool of container instances which are ready to go...

  smallcb1, smallcb2, smallcb3, smallcb4, etc.

DONE: how about having longer-running instances
that hang around more than a single request,
which are all single-node / no rebalance / no XDCR,
all for better developer tire-kicking? e.g.,

  DONE: per-request (uber)
    container instance reset/recycled after every request.
    similar to https://www.tutorialspoint.com/compile_jdbc_online.php

  DONE: multi-request (zipcar / e.g., hourly rental)
    container instance has an associated session UUID,
      and is reset/recycled only after the
      session reaches timeout from inactivity.
      data is deleted after session times out,
        and container instance is wiped / restarted.
    with user / password / sessionId generated a'la UUID,
    with network ingres/egress that's enough
      to allow for web-UI and client SDK access
      over the internet, including cbbackup/restore.
    similar to katacoda.

----------------
diagnosis links

http://www.brendangregg.com/ebpf.html

# echo "deb [trusted=yes] https://repo.iovisor.org/apt/xenial xenial-nightly main" | \
    sudo tee /etc/apt/sources.list.d/iovisor.list
# sudo apt-get update
# sudo apt-get install bpfcc-tools	# or the old package name: bcc-tools

--------------------------
docker + couchbase examples

https://hub.docker.com/_/couchbase/

https://github.com/couchbase/docker/blob/master/enterprise/couchbase-server/6.5.1/Dockerfile

https://github.com/couchbaselabs/project-avengers/tree/master/docker

https://github.com/couchbaselabs/project-avengers/blob/master/docker/couchbase/Dockerfile

https://github.com/couchbaselabs/project-avengers/blob/master/docker/cbc_pillowfight_container/Dockerfile

https://github.com/couchbaselabs?q=sequoia&type=&language=

https://github.com/couchbaselabs/sequoia/tree/master/containers/sdk

https://github.com/couchbaselabs/sequoia/blob/master/containers/sdk/Dockerfile

https://github.com/couchbaselabs/sequoia/blob/master/containers/catapult/Dockerfile - adoptopenjdk/openjdk12:latest

https://github.com/couchbaselabs/sequoia-provision

cbdyncluster
https://github.com/couchbaselabs/cbdynclusterd - daemon used to manage the test-cluster system managed by the SDKQE team.  It exposes a REST API and allows you to allocate/deallocate clusters inside of the corporate network for the purposes of doing testing.

https://github.com/couchbaselabs/cbdyncluster - This is a CLI tool for interfacing with a running cbdynclusterd.
> cbdyncluster allocate --num-nodes 3 --server-version 5.5.0
a004f847
> cbdyncluster ps
a004f847 [Owner: brett@couchbase.com, Creator: brett@couchbase.com, Timeout: 59m48s]
  dac14ff7eab9      node_1               5.5.0      172.23.111.210
  be8473e4b4d4      node_2               5.5.0      172.23.111.209
  72e31d4e0629      node_3               5.5.0      172.23.111.208
> cbdyncluster connstr a004f847
couchbase://172.23.111.210,172.23.111.209,172.23.111.208
> cbdyncluster rm a004f847

--------------------------
More diagnosis tools...

see: https://news.ycombinator.com/item?id=24341867

- atop (great for finding out what's causing system-wide slowness when
  you're not sure whether it's CPU/disk/network/temperature/etc.)

- iotop/iftop/sar (top equivalents for disk IO, network traffic, and
  sysstat counters)

- glances/nmon/dstat/iptraf-ng (pretty monitoring CLI-GUI utils with
  more color)

- docker stats (htop equivalent for docker containers)

Joining a tools/diagnostic container
  to a container that you're about to run...
  In docker, it's done by passing --pid=container:$TARGETCONTAINER to docker run
  See: https://docs.docker.com/engine/reference/run/#pid-settings--...

--------------------------
https://github.com/StepicOrg/epicbox
Run untrusted code in secure Docker based sandboxes...
A Python library to run untrusted code in secure, isolated Docker
based sandboxes. It is used to automatically grade programming
assignments on Stepik.org.

---------
http://stealth.openwall.net/xSports/shocker.c
Old - shows how to escape docker container instance.

---------
https://github.com/genuinetools/bane
Custom & better AppArmor profile generator for Docker containers.

---------
https://github.com/docker/docker-bench-security
The Docker Bench for Security is a script that checks for dozens of common
best-practices around deploying Docker containers in production.

---------
https://security.stackexchange.com/questions/107850/docker-as-a-sandbox-for-untrusted-code
From 2015.

---------
https://github.com/vaharoni/trusted-sandbox
Run untrusted code in a contained sandbox, using Docker
from 2014

---------
pgrep -u couchbase -a -n
  and then can killall by process group
  killall -g pgrp
  pkill -g pgrp

---------
Information about Cookie Pro (OptanonWrapper)
  https://community.cookiepro.com/s/article/UUID-730ad441-6c4d-7877-7f85-36f1e801e8ca

---------
Multi-container playground tours...

Mockup (slides) with links to UX testing feedback (2021/03) - https://docs.google.com/presentation/d/1UFItkNOFp7BYKuzWBEGN5x8otxbGykdaOHW6cFbZZN0/edit?pli=1#slide=id.gcb03ecc9b0_0_6

Walkthru (recording) of internals of the multi-container playground with Eric B. (2021/03/29) - https://couchbase.zoom.us/rec/play/tZUxpjfd9ZzOdvNVVZIEzEXxKZfBd6N-QaYlIY3VnU-hC5Xtvg_h8QfmKmY6sg5se8rvyjinVc2Leb7z.YB_nMwLv983kb0rl?startTime=1617042768000

Another walkthru (recording) of multi-container playground internals with David Quintas Vilanova (2021/04/19) - Topic: multi-container playground internals, w/ David Quintas Vilanova - https://couchbase.zoom.us/rec/share/WBlGcJqrLxjgvaeVLuFQ_WhVx4o1i_hQNQoiNOTyYNTMnNGqpwZ7WQg8jfUVCOu_.65d7w3TmcOhw3AMV / Access Passcode: 62.6z&+b

To play with a multi-container playground tour, such as the
tours-multi/first-xdcr tour...

http://localhost:8080/session?groupSize=2&bodyClass=normal&e=static/tours-multi/aaa.html%3Fname%3Dfirst-xdcr&title=Try+Data+Streaming+with+Couchbase...&init=tours-multi/shs&defaultBucket=NONE

For example...

http://couchbase.live/session?groupSize=2&bodyClass=normal&e=static/tours-multi/aaa.html%3Fname%3Dfirst-xdcr&title=Try+Data+Streaming+with+Couchbase...&init=tours-multi/shs&defaultBucket=NONE



