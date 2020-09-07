package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var CBAdminPassword = "" // Initialized by CB_ADMIN_PASSWORD env.
var CBAdminPasswordDefault = "small-house-secret"

// -----------------------------------

// Channel of container instance #'s that are ready.
var readyCh chan int

// Channel of container instance restart requests.
var restartCh chan Restart

// -----------------------------------

// The user for docker exec.
var ExecUser = "couchbase:couchbase"

// Map from lang (or code file suffix) to execPrefix (exec command
// prefix for executing code).
var ExecPrefixes = map[string]string{
	"java": "/run-java.sh",
}

// -----------------------------------

// Port mapping of container port # to containerPublishPortBase + delta.
var PortMapping = [][]int{
	[]int{8091, 1}, // 8091 is exposed on port 10000 + 1.
	[]int{8092, 2}, // 8092 is exposed on port 10000 + 2.
	[]int{8093, 3},
	[]int{8094, 4},
	[]int{8095, 5},
	[]int{8096, 6},

	[]int{18091, 11}, // 18091 is exposed on port 10000 + 11.
	[]int{18092, 12}, // 18092 is exposed on port 10000 + 12.
	[]int{18093, 13},
	[]int{18094, 14},
	[]int{18095, 15},
	[]int{18096, 16},

	[]int{11207, 27}, // 11207 is exposed on port 10000 + 27.
	[]int{11210, 30}, // 11210 is exposed on port 10000 + 30.
	[]int{11211, 31}, // 11211 is exposed on port 10000 + 31.
}

var PortMap = map[int]int{}

func init() {
	for _, pair := range PortMapping {
		PortMap[pair[0]] = pair[1]
	}
}

// ------------------------------------------------

func main() {
	StatsInfo("main.startTime",
		time.Now().Format("2006-01-02T15:04:05.000-07:00"))

	StatsInfo("main.args", strings.Join(os.Args, " "))

	flag.Parse()

	if *h || *help {
		flag.Usage()
		os.Exit(2)
	}

	var flags []string
	flag.VisitAll(func(f *flag.Flag) {
		flags = append(flags, fmt.Sprintf("%s=%v", f.Name, f.Value))
	})

	StatsInfo("main.flags", strings.Join(flags, " "))

	CBAdminPassword = os.Getenv("CB_ADMIN_PASSWORD")
	if CBAdminPassword == "" {
		CBAdminPassword = CBAdminPasswordDefault
	}

	// The readyCh and restartCh are created with capacity
	// equal to the # of containers to lower the chance of
	// client requests and restarters from having to wait.

	readyCh = make(chan int, *containers)

	restartCh = make(chan Restart, *containers)

	// Spawn the restarter goroutines.
	for i := 0; i < *restarters; i++ {
		go Restarter(i, restartCh,
			*containerPublishAddr,
			*containerPublishPortBase,
			*containerPublishPortSpan,
			PortMapping)
	}

	// Restart the required # of containers.
	for containerId := 0; containerId < *containers; containerId++ {
		restartCh <- Restart{
			ContainerId: containerId,
			ReadyCh:     readyCh,
		}
	}

	go SessionsChecker(*sessionsCheckEvery, *sessionsMaxAge)

	mux := http.NewServeMux()

	HttpMuxInit(mux)

	go HttpProxy(*listenProxy, *proxyFlushInterval,
		*containerPublishHost,
		PortMap,
		*containerPublishPortBase,
		*containerPublishPortSpan)

	log.Printf("INFO: main, listen: %s", *listen)

	log.Fatal(http.ListenAndServe(*listen, mux))
}
