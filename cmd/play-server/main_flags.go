package main

import (
	"flag"
	"time"
)

var (
	h = flag.Bool("h", false, "print help/usage and exit")

	codeDuration = flag.Duration("codeDuration", 20*time.Second,
		"duration that a client's request code may run on an assigned container instance")

	codeMaxLen = flag.Int("codeMaxLen", 10000,
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

	feedbackURL = flag.String("feedbackURL", "https://devportal-api.prod.couchbase.live/pageLikes",
		"URL to send the feedback from pages")

	host = flag.String("host", "127.0.0.1",
		"host that the service will be publically available as")

	jsFlags = flag.String("jsFlags", "",
		"optionally disable JS script inclusion (values: allOff, analyticsOff, optanonOff)")

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
		15*time.Minute,
		"duration by inactivity for which sessions are automatically exited")

	sessionsCheckEvery = flag.Duration("sessionsCheckEvery",
		30*time.Second,
		"duration or interval to wait before checking sessions")

	staticDir = flag.String("staticDir", "cmd/play-server/static",
		"path to the 'static' resources directory")

	statsEvery = flag.Duration("statsEvery",
		20*time.Second,
		"duration or interval between grabbing another sample of stats")

	targetsCookieName = flag.String("targetsCookieName",
		"capella",
		"cookie name to store couchbase targets information")

	targetsMaxAge = flag.Duration("targetsMaxAge",
		30*24*time.Hour,
		"duration by age for which couchbase targets are automatically not used")

	version = flag.String("version", "tmp/ns_server.app.vsn",
		"version string or filename that holds version string")

	encryptKey = flag.String("encryptKey", "12345678901234567890123456789012",
		"secret key for encrypted text")

	natPublicIP = flag.String("natPublicIP", "50.18.183.235", "NAT public IP")

	natPrivateIP = flag.String("natPrivateIP", "10.0.1.111", "NAT private IP")

	baseUrl = flag.String("baseUrl", "couchbase.live",
		"base url to use as the link to homepage")

	tlsServer = flag.Bool("tlsServer", false,
		"TLS mode for server")

	tlsTerminal = flag.Bool("tlsTerminal", false,
		"TLs mode for Terminal")

	tlsTerminalProxy = flag.Bool("tlsTerminalProxy", false,
		"TLs mode for Terminal proxy")

	tlsListenPortBase = flag.Int("tlsListenPortBase", 20000,
		"base or starting port # for container instances with tls mode")

	tlsKey = flag.String("tlsKey", "certs/server.key",
		"path to the TLS key file")

	tlsCert = flag.String("tlsCert", "certs/server.crt",
		"path to the TLS certificate file")

	serverTools = flag.String("serverTools", "couchbase-server couchdb gometa goport "+
		"gosecrets goxdcr gozip cbas cbft cbft-bleve backup indexer install "+
		"eventing-consumer eventing-producer "+
		"memcached projector prometheus mobile-service "+
		"couch_view_file_merger couch_view_group_cleanup couch_view_group_compactor "+
		"couch_view_index_builder couch_view_index_updater couchfile_upgrade cbq-engine couchjs ct_run "+
		"vbmap cbupgrade saslauthd-port sigar_port "+
		"c_rehash pcre-config pcregrep pcretest ",
		"List of server tools to hide for the cli tools")

	cliTools = flag.String("cliTools", "cbbackupmgr cbq cbc cbimport cbexport "+
		"cbc-pillowfight mctimings mcstat cbstats ",
		"List of visible cli tools")
)
