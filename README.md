Dependencies...

* golang
* docker
* make

Instructions for use...

One time setup/init/build steps...

    # 1) To create the docker image 'smallcb',
    # which includes both couchbase-server & couchbase sdks...
    make build

    # 2) Temporarily launch a smallcb docker image from step 1
    # that initializes a couchbase-server with sample data
    # and configured with lower resource utilization, in order
    # to create the reusable vol-snapshot subdirectory...
    make create

    # 3) Compile the web/app server...
    make play-server

And, to start the web/app server...

    # Listens on port 8080 for web browser requests,
    # where some warmup time is needed (as it launches
    # container instances via 'make restart' invocations
    # that are based on the vol-snapshot)...
    ./play-server

For command-line usage/help...

    ./play-server -h

-------------------------
Aside...

    # To create the docker image 'smallcb-sdks',
    # which includes only the couchbase sdks...
    make IMAGE_FROM=base IMAGE_NAME=smallcb-sdks build

-------------------------
TODO's...

figure out where to run this in production?

how about staging?

roughly, how much will it cost?

need to su to couchbase to install java SDK, nodejs SDK, etc?

first end-to-end demo on laptop?

first end-to-end demo on cloud (staging)?

first examples from not-steve?

need to golang proxy to use the remapped port #'s?
  in the REST responses, to rewrite REST json maps
  to list server hostnames/addrs correctly?

docker container needs to -p or publish/expose ports
  on 0.0.0.0 addr instead of 127.0.0.1 addr?
  See "-containerPublishAddr" cmd-line flag.

create couchbase user better than Administrator:password,
  especially dynamically with user & password that look
  more like UUID's when we're in >= zipcar mode?

client-side JS to prevent over-clicking on Run/Submit button,
  where the current attempt of onRunSubmit()
  doesn't work very well for the 2nd click on 'run'.

client-side JS to prevent Back Button or navigating away
  from losing work in the code textarea?

client-side code syntax highlighting / styling via
  codemirror, ACE code editor?

couchbase style header / footer?
  call-to-action / download couchbase?
  link to docs?

CSS styling for mobile / narrow screens?

better 404 error screen?

when there are enough examples, use a tree-control
  on the left-hand-side with a scrollable panel,
  perhaps with mouseovers with longer explanations?

when the page is scrolled down, and you click on another
  example link, there's a disconcerting jump back to
  top of the page rather than having the page stay
  mostly stable -- similar to docs behavior?
  maybe use an iframe, but then URL won't be bookmarkable?

favorites / recommended examples?

some examples that only make sense when there's
  a longer-running session >= zipcar mode?

feedback comments or votes on examples?

capture email to get a longer-running
  dev tire-kicking instance?
  full name?  password?
  CAPTCHA?

can i have >1 longer running instance per email?

what if my email and/or name are already used?
  if so, can i get another zipcar for me?

where do we store email to instance UUID info?

sizing?
  some rough disk usage info...

    % du -s -h vol-0        ==> 8.1M vol-0
    % du -s -h vol-snapshot ==> 5.6M vol-snapshot

google analytics?

stats?
  keep average time of restarts, for fake/estimated progress/ETA bars?

DNS, ELB & subdomains?

use nginx proxy for subdomain based routing?
  See, the following, but which unfortunately doesn't seem to handle
  multiple virtual ports...
  https://blog.florianlopes.io/host-multiple-websites-on-single-host-docker/

can docker checkpoint (experimental feature in docker)
  help speed up slow restart times?

should we use docker build env vars?

use docker on docker?

use docker networking features?
  use docker network overlay --internal mode?
  perhaps too complex.

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

SECURITY: docker build can set ulimits
  and optional security-opts?
  See:
    https://docs.couchbase.com/server/current/install/best-practices-vm.html
    docker run -d --ulimit nofile=40960:40960 \
      --ulimit core=100000000:100000000 \
      --ulimit memlock=100000000:100000000 \
      --name db -p 8091-8096:8091-8096 -p 11210-11211:11210-11211 couchbase

SECURITY: only allow host to connect (or proxy)?

SECURITY: cpu/memory usage limits?

SECURITY: restart the host system every day?
  just in case that unsafe code escapes
  the container sandbox via kernel hack?

SECURITY: hosting IAM rules?

SECURITY: RBAC to limit access?

SECURITY: need a CAPTCHA?

SECURITY: need spam/flood throttling?

copy/pastable connection snippets for popular languages
  and SDK's?
  for >= zipcar mode?

need 1 or more test users / test examples / test container instances?

need ping / sanity checking REST endpoints?

iframe for access to web admin portal?
  need server-side proxy in golang?
  perhaps access to just query workbench?
  ns-server does not like iframes, so need header rewrites?

or pop up web admin portal in separate tab?
  with rewrites / injection of headline messages
  or advertisements?

should have a one-click workload generator?

how about having longer-running instances
that hang around more than a single request,
which are all single-node / no rebalance / no XDCR,
all for better developer tire-kicking?
e.g.,
  per-request (uber)
    container instance reset/recycled after every request.
    similar to https://www.tutorialspoint.com/compile_jdbc_online.php

  multi-request (zipcar / hourly rental)
    container instance has an associated session UUID,
      and is reset/recycled only after the
      session times out from inactivity, or
      from a too-long session (might be a robot, not a human).
    data is deleted after session times out.
    and, user / password has to be generated UUID?
      and network ingres/egress that's enough,
      intended to allow for cbbackup/restore from elsewhere?
    similar to katacoda?

  multi-request-with-data-freezing/thawing (hertz/avid, multi-day rental)
    after a timeout from inactivity,
    the data is snapshotted and parked in quiescent garage somewhere...
      like on local disk,
           or onto S3.
    when the user comes back, data is thawed,
      against a restarted container,
      perhaps at a different assigned host:port?
      which takes some time (e.g., go get a coffee) while defrosting?

  finally, if you want a lease of 1 or more fleet of cars (clustering),
    with attached pool of hotel chauffeurs and
    mechanic/maintenance services...
    then use Couchbase Cloud.

-------------------------
On new CB version release...

Does that mean a new EC2 instance?

What about frozen data --
  do we thaw them on demand, as requested?
  eventually give up on versions that's too old?

How about on data that is super old?

GDPR with emails and PII?

-------------------------
examples can now be collected into separate "books"?
  multiple 'examples' directories are now supported.

InfoBefore / InfoAfter can now have HTML markup,
  like links to relevant docs page or "next" step links.

-------------------------
# Security

- timeouts for long-running programs, see:
  codeMaxDuration and containerWaitDuration.

- docker exec as -u couchbase:couchbase (user:group), not as root.

-------------------------
use cases
  try it now
    open-ended tire kicking?
    of SDK / API testing?
       and/or N1QL?
    try-it-now buttons in the docs & tutorials?

more use cases with persistent data?
  CI/CD tests?
  backend jobs on-demand?
    analytics?
    quick slice/dice jobs against big data (covid19)?
    AI/ML jobs?

    serverless event processing?

dev-mode config is reusable for laptops, too?

--------------------------
handwave design ideas...

couchbase CB_BASE_VER docker

  tweak & configure couchbase to lower TCO
  init couchbase
  tweak & configure couchbase to lower TCO, part 2
  create buckets
  load sample data

  --------------------

  add language 1 tools
  add sdk 1 V1 stuff
  add sdk 1 V2 stuff

  add language N tools
  add sdk N V1 stuff
  add sdk N V2 stuff

  then, freeze or snapshot /opt/couchbase/var
    as a good restart point

--------------------------
also, put faster changing stuff into host filesystem
  for easy github updates?
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
    communicates via stdin/stdout
      perhaps using "docker attach CONTAINER"
    when done, shuts down the container instance
      and restarts it cleanly (ahead of time),
        in preparation for next request,
        to reduce cold start window?
      starts couchbase, too,
        depending if we're using a local couchbase

perhaps can docker pause/unpause to reduce footprint?

--------------------------
have a pool of container instances which are ready to go...

  db-1, db-2, db-3, db-4, etc?

  perhaps write a file on whether db-X is ready to use?

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

couchbaselabs/sequoia-provision

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

Joining a tools/diagnostic container to container you're about to run...
  In docker, it's done by passing --pid=container:$TARGETCONTAINER to docker run
  See: https://docs.docker.com/engine/reference/run/#pid-settings--...

