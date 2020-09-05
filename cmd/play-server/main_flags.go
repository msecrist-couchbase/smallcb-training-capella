package main

import "flag"
import "time"

var (
	h = flag.Bool("h", false, "print help/usage and exit")

	help = flag.Bool("help", false, "print help/usage and exit")

	codeMaxLen = flag.Int("codeMaxLen", 16000,
		"max length of a client's request code in bytes")

	codeDuration = flag.Duration("codeDuration", 10*time.Second,
		"duration that a client's request code may run on an assigned container instance")

	containerNamePrefix = flag.String("containerNamePrefix", "smallcb-",
		"prefix of the names of container instances")

	containerVolPrefix = flag.String("containerVolPrefix", "vol-instances/vol-",
		"prefix of the volume directories of container instances")

	containerPublishHost = flag.String("containerPublishHost", "try.couchbase.dev",
		"host to use for generating connection URL's")

	containerPublishAddr = flag.String("containerPublishAddr", "127.0.0.1",
		"addr for publishing container instance ports")

	containerPublishPortBase = flag.Int("containerPublishPortBase", 10000,
		"base or starting port # for container instances")

	containerPublishPortSpan = flag.Int("containerPublishPortSpan", 100,
		"number of port #'s allocated for each container instance")

	containerWaitDuration = flag.Duration("containerWaitDuration", 20*time.Second,
		"duration that a client's request will wait for a ready container instance")

	containers = flag.Int("containers", 1,
		"# of container instances")

	restarters = flag.Int("restarters", 1,
		"# of restarters of the container instances")

	maxCaptchas = flag.Int("maxCaptchas", 50,
		"# of captcha guesses to keep")

	staticDir = flag.String("staticDir", "cmd/play-server/static",
		"path to the 'static' directory")

	listen = flag.String("listen", ":8080",
		"HTTP listen [addr]:port")

	sessionsMaxAge = flag.Duration("sessionsMaxAge",
		5*time.Minute,
		"exit sessions older than this duration")

	sessionsCheckEvery = flag.Duration("sessionsCheckEvery",
		30*time.Second,
		"duration to wait before checking sessions")
)
