package main

import "flag"
import "time"

var (
	h = flag.Bool("h", false, "print help/usage and exit")

	codeDuration = flag.Duration("codeDuration", 20*time.Second,
		"duration that a client's request code may run on an assigned container instance")

	codeMaxLen = flag.Int("codeMaxLen", 8000,
		"max length of a client's request code in bytes")

	containerNamePrefix = flag.String("containerNamePrefix", "smallcb-",
		"prefix of the names of container instances")

	containerVolPrefix = flag.String("containerVolPrefix", "vol-instances/vol-",
		"prefix of the volume directories of container instances")

	containerPrepDuration = flag.Duration("containerPrepDuration", 400*time.Millisecond,
		"duration to sleep to allow container setup or preparation to finish")

	containerWaitDuration = flag.Duration("containerWaitDuration", 20*time.Second,
		"duration that a client's request will wait for a ready container instance")

	containers = flag.Int("containers", 1,
		"# of container instances")

	containersSingleUse = flag.Int("containersSingleUse", 0,
		"# of container instances to keep as single use or session-less")

	host = flag.String("host", "127.0.0.1",
		"host that the service will be publically available as")

	listen = flag.String("listen", ":8080",
		"[addr]:port for play-server's web UI / REST API")

	listenPortBase = flag.Int("listenPortBase", 10000,
		"base or starting port # for container instances")

	listenPortSpan = flag.Int("listenPortSpan", 100,
		"span or range of port #'s allocated to each container instance")

	listenProxy = flag.String("listenProxy", ":8091,:8093",
		"[addr]:port(s) that will be proxied to container instances")

	maxCaptchas = flag.Int("maxCaptchas", 50,
		"# of captcha guesses to keep")

	proxyFlushInterval = flag.Duration("proxyFlushInterval",
		200*time.Millisecond,
		"duration before flushing http proxy response streams")

	restarters = flag.Int("restarters", 1,
		"# of restarters of the container instances")

	sessionsMaxAge = flag.Duration("sessionsMaxAge",
		10*time.Minute,
		"duration by age for which sessions are automatically exited")

	sessionsMaxIdle = flag.Duration("sessionsMaxIdle",
		5*time.Minute,
		"duration by inactivity for which sessions are automatically exited")

	sessionsCheckEvery = flag.Duration("sessionsCheckEvery",
		30*time.Second,
		"duration or interval to wait before checking sessions")

	staticDir = flag.String("staticDir", "cmd/play-server/static",
		"path to the 'static' resources directory")

	statsEvery = flag.Duration("statsEvery",
		20*time.Second,
		"duration or interval between grabbing another sample of stats")

	version = flag.String("version", "tmp/ns_server.app.vsn",
		"version string or filename that holds version string")
)
