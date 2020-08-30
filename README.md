Instructions for use...

One time setup/init/build steps...

    # 1) Create the docker image (smallcb)...
    make build

    # 2) Use the smallcb docker image from step 1 to
    # initialize a couchbase server with sample data,
    # to create the reusable vol-snapshot subdirectory...
    make create

    # 3) Compile the web/app server...
    make play-server

And, to start the web/app server...

    # Listens on port 8080 for web browser requests,
    # where some warmup time is needed (as it launches
    # container instances via 'make restart' invocations)...
    ./play-server

For command-line usage/help...

    ./play-server -h

-------------------------
TODO's...

figure out where to run this in production?

how about staging?

roughly, how much will it cost?

need to su to couchbase to install java SDK, nodejs SDK, etc?

first end-to-end demo on laptop?

first end-to-end demo on cloud (staging)?

need to golang proxy to use the remapped port #'s?
  in the REST responses, to rewrite REST json maps
  to list server hostnames/addrs correctly.

docker container needs to -p or publish/expose ports on 0.0.0.0 addr
  instead of 127.0.0.1 addr?

create couchbase user better than Administrator:password,
  especially with user & password that look more like
  UUID's when we're in >= zipcar mode?

client-side JS to prevent over-clicking on Run/Submit button?

client-side JS to prevent Back Button or navigating away
  from losing work?

client-side code syntax highlighting / styling / codemirror, ACME code editor?

CSS styling / mobile?

when there are enough examples, use a tree-control
  on the left-hand-side with a scrollable panel,
  perhaps with mouseover longer explanations?

capture email to get a longer-running
  dev tire-kicking instance?

some examples only make sense with a longer-running
  dev tire-kicking instance?

can i have >1 longer running instance per email?

what if my email is already used?

where do we store email to instance UUID info?

some rough disk usage info...

    % du -s -h vol-0
    8.1M vol-0
    % du -s -h vol-snapshot
    5.6M vol-snapshot

google analytics?

DNS & subdomains?

can docker checkpoint (experimental feature)
  help speed up slow restart times?

use docker build env vars?

use tmpfs for faster restarts and less real i/o?

use docker on docker?

docker run has interesting tweakable limits?

docker run --read-only flag?

docker run --tmpfs flag?

use nginx proxy for subdomain based routing?
  See, the following, but which unfortunately doesn't seem to handle
  multiple virtual ports...
  https://blog.florianlopes.io/host-multiple-websites-on-single-host-docker/

use docker network overlay --internal mode?
  https://www.reddit.com/r/docker/comments/hvs7n9/how_do_i_prevent_a_container_from_making_outgoing/
  If your container is hosted on a VM in Azure, AWS, GCP, OpenStack
  etc, you'll want to restrict Egress (outgoing) traffic from that VM
  via Security Rules on the private network

SECURITY: docker build can set ulimits
  and optional security-opts?

SECURITY: turn off egress networking?

SECURITY: only allow host to connect (or proxy)?

SECURITY: cpu/memory usage limits?

SECURITY: restart the host system every day?
          unsafe code can escape the container sandbox via kernel hack?

SECURITY: RBAC to limit access?

SECURITY: need a CAPTCHA?

iframe for access to web admin portal?
  need server-side proxy in golang?
  access to just query workbench?

or pop up web admin portal in separate tab?
  with rewrites / injection of messages?

how about having longer-running instances
that hang around more than a single request,
which are all single-node / no rebalance / no XDCR?
e.g.,
  per-request (uber)
    container instance reset/recycled after every request.

  multi-request (zipcar / hourly rental)
    container instance has an associated session UUID,
      and is reset/recycled only after the
      session times out from inactivity, or
      from a too-long session (might be a robot, not a human).
    data is deleted after session times out.
    and, user / password has to be generated UUID?
      and network ingres/egress that's enough,
      intended to allow for cbbackup/restore from elsewhere?

  multi-request-with-data-freezing/thawing (hertz/avid, multi-day rental)
    after a timeout from inactivity,
    the data is snapshotted and parked in
      long-term-parking somewhere...
      like on local disk,
           or onto S3.
    when the user comes back, data is thawed,
      against a restarted container,
      which takes some time (e.g., go get a coffee).

if you want a lease of 1 car or more (clustering),
  with attached chauffering & mechanics services...
  use Couchbase Cloud.

should there be a workload generator included?

-------------------------
On new CB version release...

Does that mean a new EC2 instance?

What about frozen data --
  do we thaw them on demand, as requested?
  eventually give up on versions that's too old?

-------------------------
# Security

- timeouts for long-running programs, see:
  codeMaxDuration and workersMaxDuration.

- docker exec as couchbase:couchbase

-------------------------
use cases
  try it now
    open-ended tire kicking?
    buttons in the docs & tutorials?

use cases with persistent data?
  CI/CD tests?
  backend jobs on-demand?
    analytics?
    quick slice/dice jobs against big data (covid19)?
    AI/ML jobs?
    serverless event processing?

dev-mode config?

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

  write a file on whether db-X is ready to use?

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
