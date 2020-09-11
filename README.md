Dependencies...

* golang
  * tip: after checking out this project, run "go get ./..."
    to download golang dependencies.
    * tip: you might need to setup your GOPATH env variable.
* docker
* make

Instructions for use...

One time setup/init/build steps...

    # 1) To create the docker image 'smallcb',
    # which includes both couchbase-server & couchbase sdks...
    make build

    # 2) Temporarily launch a smallcb docker image from step 1
    # that initializes a couchbase-server with sample data
    # and configures it for lower resource utilization.
    #
    # This step creates and populates the vol-snapshot directory.
    #
    # TIP: Make sure you're not already running couchbase-server
    # as it needs the standard couchbase-server port #'s.
    make create

    # 3) Compile the web/app server...
    make play-server

And, to start the web/app server...

    # Listens on port 8080, 8091 and 8093 for web browser requests...
    #
    # TIP: Make sure you're not already using those ports,
    # or use the command-line flags to change the port #'s.
    ./play-server

For command-line usage/help...

    ./play-server -h

Example usage during development...

    ./play-server -sessionsMaxAge 20m -containers 2 -restarters=2

Production usage should set the CB_ADMIN_PASSWORD env
variable for security and the host parameter...

    CB_ADMIN_PASSWORD=secret-here ./play-server -host try.couchbase.dev

-------------------------
Aside...

    # To create the docker image 'smallcb-sdks',
    # which includes only the couchbase sdks...
    make IMAGE_FROM=base IMAGE_NAME=smallcb-sdks build

-------------------------
TODO's...

figure out where to run this in production?

how about staging?

need a "down for maintenance" static web page path or toggle?

roughly, how much will it cost?

need to su to couchbase to install java SDK, nodejs SDK, etc?

need a cgroup or a throwaway container to safely run
  submitted code with hard timeouts / resource limits?

show & tell?

first end-to-end demo on laptop?

first end-to-end demo on cloud (staging)?

first examples from not-steve?

lots of examples are auto-scraped from docs and
  put into: https://github.com/couchbaselabs/sdk-examples
    every once in awhile.
  see also: https://github.com/couchbaselabs/devguide-examples
    which are older.
  ask Matt I. for more details

client-side JS to prevent over-clicking on Run/Submit button,
  where the current attempt of onRunSubmit()
  doesn't work very well for the 2nd click on 'run'.

client-side JS to prevent Back Button or navigating away
  from losing work in the code textarea?

client-side code syntax highlighting / styling via
  codemirror, ACE code editor?

couchbase style header / footer?
  call-to-action / download couchbase?

CSS styling for mobile / narrow screens?

better 404 error screen?

when there are enough examples, use a tree-control
  on the left-hand-side with a scrollable panel,
  perhaps with mouseovers with longer explanations?

favorites / recommended examples?  starred?

some examples that only make sense when there's
  a longer-running session >= zipcar mode?

feedback comments or votes on examples?

CAPTCHA random seed looks weak
  seeing repeats on process start?

CAPTCHA panics sometimes at rand.Intn()?

can i have >1 longer running instance per email?
  ANS: currently, sorta -- full name + email must be unique.

what if my email and/or name are already used?
  if so, can i get another zipcar for me?
  ANS: currently, sorta -- full name + email must be unique.

store logs on S3?

google analytics?

stats?
  keep average time of restarts, for fake/estimated progress/ETA bars?
  do we dump stats to logs or S3 occasionally?

DNS, ELB & subdomains?

ELB and sticky load balancing?

health watchers and elastic scaling -- bring up more nodes
  when there's more people and auto-scale-down when
  traffic goes away?

sizing?
  some rough disk usage info...

    % du -s -h vol-0        ==> 8.1M vol-0
    % du -s -h vol-snapshot ==> 5.6M vol-snapshot

inject better UI into web admin UI?

proper web terminal UI?

docker on mac OSX sometimes gets 'stuck' -- container
  instances aren't restartable sometimes?  Will this be the
  case also on linux?  If so, perhaps need a "kill the
  entire server/machine and replace it" button?

should we use docker build env vars?

should we use docker on docker?

use docker networking features?
  use docker network overlay --internal mode?
  perhaps too complex?
  perhaps necessary if we want a play run-only container for submitted code?

use tmpfs for faster restarts and less real i/o,
  at the cost of RAM?
  docker run --tmpfs flag?

docker run has interesting tweakable runtime resource limits
  to look at?

docker run --read-only flag?

SECURITY: turn off egress networking?
  https://www.reddit.com/r/docker/comments/hvs7n9/how_do_i_prevent_a_container_from_making_outgoing/
  If your container is hosted on a VM in Azure, AWS, GCP, OpenStack
  etc, you'll want to restrict Egress (outgoing) traffic
  or new outbound connections from the host
  via Security Rules on the private network?

SECURITY: remove ability to strace in production?
  CVE-2014-4699: A bug in ptrace() could allow privilege
  escalation. Docker disables ptrace() inside the container using
  apparmor, seccomp and by dropping CAP_PTRACE.

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
  the container sandbox via kernel hack?

SECURITY: hosting IAM rules?

SECURITY: RBAC to limit access?

SECURITY: need a CAPTCHA?

SECURITY: need spam/flood throttling?

SECURITY: need a bad-list of emails that we don't like?

copy/pastable connection snippets for popular languages
  and SDK's, for >= zipcar mode?

need 1 or more test users / test examples / test container instances?

iframe for access to web admin portal?
  need server-side proxy in golang?
  perhaps access to just query workbench?
  DONE: ns-server does not like iframes, so the proxy removes
    the X-Frame-Options DENY header from the response.

the proxy serving of web-admin UI login screen
  always goes to container 0 -- but what if
  container 0 is restarting?  Then the UI login screen
  can't be served?  Maybe play-server's proxy should cache?

or pop up web admin portal in separate tab?

popup tours in injection?
  https://kamranahmed.info/driver.js/

output (stdout / stderr) is not streaming?

how about having longer-running instances
that hang around more than a single request,
which are all single-node / no rebalance / no XDCR,
all for better developer tire-kicking?
e.g.,
  DONE: per-request (uber)
    container instance reset/recycled after every request.
    similar to https://www.tutorialspoint.com/compile_jdbc_online.php

  DONE: multi-request (zipcar / e.g., hourly rental)
    container instance has an associated session UUID,
      and is reset/recycled only after the
      session reachs timeout from inactivity.
    data is deleted after session times out.
    and, user / password is generated a'la UUID,
    with network ingres/egress that's enough
      to allow for cbbackup/restore from elsewhere.
    similar to katacoda.

  multi-request-with-data-freezing/thawing (hertz/avid, multi-day rental)
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

UI should show the CB version?

Does a new CB version mean a new EC2 instance
  and then redirect the DNS / ELB,
  a'la rolling upgrade?

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

dev-mode config is reusable for laptops, too?

dev-mode still asking for stats too much?

dev-mode, phase II, needs core product improvements?

--------------------------
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

DONE: examples can now be collected into separate "books"?
  multiple example subdirectories are now supported.

DONE: InfoBefore / InfoAfter can now have HTML markup,
  like links to relevant docs page or "next" step links.

DONE: one-click workload generator, done via cbworkloadgen.

DONE: create couchbase user better than Administrator:password,
  especially dynamically with user & password that look
  more like UUID's when we're in >= zipcar mode.

DONE: change the Administrator password in 'make create'.

DONE: using docker --add-host to add to /etc/hosts file so that
  couchbase://try.couchbase.dev:8091 connection string works.

DONE: need to golang proxy to use the remapped port #'s?
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

--------------------------
handwave design ideas...

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
also, put faster changing stuff into host filesystem
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
it might connect to localhost couchbase,
 or might connect to remotehost couchbase,
   for longer persistence?

--------------------------
on the host
  web/app-server
    which starts container
      (e.g., docker run -rm SAFE SAFE whatever),
    communicates via stdin/stdout,
      perhaps using "docker attach CONTAINER"?
    when done, shuts down the container instance
      and restarts it cleanly (ahead of time),
        in preparation for next request,
        to reduce cold start window?

perhaps can use docker pause/unpause feature to reduce footprint?

--------------------------
also have a pool of container instances which are ready to go...

  smallcb1, smallcb2, smallcb3, smallcb4, etc.

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
  more colors)

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
