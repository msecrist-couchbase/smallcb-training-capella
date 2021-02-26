package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Initialized by CB_ADMIN_PASSWORD env.
var CBAdminPassword = ""

// Production deployments are strongly encouraged to set a
// CB_ADMIN_PASSWORD env variable which will be used instead
// of this default.
var CBAdminPasswordDefault = "small-house-secret"

// -----------------------------------

var StartTime time.Time

// -----------------------------------

// Channel of container instance #'s that are ready.
var readyCh chan int

// Channel of container instance restart requests.
var restartCh chan Restart

// -----------------------------------

// Map from lang (or code file suffix) to execPrefix (exec command
// prefix for executing code).
var ExecPrefixes = map[string]string{
	"go":     "/run-go.sh",
	"java":   "/run-java.sh",
	"nodejs": "/run-nodejs.sh",
	"py":     "/run-py.sh",
	"dotnet": "/run-dotnet.sh",
	"php":    "/run-php.sh",
}

// -----------------------------------

// Port mapping of container port # to listenPortBase + delta.
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

	[]int{1337, 40}, // The gritty port is exposed on port 10000 + 40.
}

var PortMap = map[int]int{}

func init() {
	for _, pair := range PortMapping {
		PortMap[pair[0]] = pair[1]
	}
}

// ------------------------------------------------

var VersionSDKs = []map[string]string{}

// ------------------------------------------------

func main() {
	StartTime = time.Now()

	StatsInfo("main.startTime", StartTime.Format(
		"2006-01-02T15:04:05.000-07:00"))

	StatsInfo("main.args", strings.Join(os.Args, " "))

	flag.Parse()

	if *h {
		flag.Usage()
		os.Exit(2)
	}

	// Couchbase Server version...
	if strings.Index(*version, "/") >= 0 {
		b, err := ioutil.ReadFile(*version)
		if err != nil {
			log.Fatalf("could not read version file: %s, err: %v",
				*version, err)
		}

		files, err := filepath.Glob(filepath.Dir(*version) + "/VERSION-sdk-*.ver")
		if err == nil {
			for _, file := range files {
				ver, err := ioutil.ReadFile(file)
				if err == nil && len(ver) > 0 {
					name := filepath.Base(file)
					name = name[:len(name)-4]
					name = name[len("VERSION-sdk-"):]

					VersionSDKs = append(VersionSDKs, map[string]string{
						"name": name,
						"ver":  string(ver),
					})
				}
			}
		}

		*version = string(b)
	}

	var flags []string
	flag.VisitAll(func(f *flag.Flag) {
		flags = append(flags, fmt.Sprintf("%s=%v", f.Name, f.Value))
	})

	StatsInfo("main.flags", strings.Join(flags, " "))

	gitDescribe, _ := ExecCmd(context.Background(),
		exec.Command("git", "describe", "--always"), 10*time.Second)

	StatsInfo("git.describe", string(gitDescribe))

	CBAdminPassword = os.Getenv("CB_ADMIN_PASSWORD")
	if CBAdminPassword == "" {
		CBAdminPassword = CBAdminPasswordDefault
	}

	listenAddr := strings.Split(*listen, ":")[0]
	listenPort, _ := strconv.Atoi(strings.Split(*listen, ":")[1])

	// The readyCh and restartCh are created with capacity
	// equal to the # of containers to lower the chance of
	// client requests and restarters from having to wait.

	readyCh = make(chan int, *containers)

	restartCh = make(chan Restart, *containers)

	// Spawn the restarter goroutines.
	for i := 0; i < *restarters; i++ {
		go Restarter(i, restartCh, *host,
			listenAddr,
			*listenPortBase,
			*listenPortSpan,
			PortMapping)
	}

	// Restart the required # of containers.
	for containerId := 0; containerId < *containers; containerId++ {
		restartCh <- Restart{
			ContainerId: containerId,
			ReadyCh:     readyCh,
		}
	}

	go SessionsChecker(*sessionsCheckEvery, *sessionsMaxAge, *sessionsMaxIdle)

	mux := http.NewServeMux()

	HttpMuxInit(mux)

	for _, lp := range strings.Split(*listenProxy, ",") {
		go HttpProxy(lp, *staticDir, *proxyFlushInterval,
			*host, listenPort, PortMap,
			*listenPortBase,
			*listenPortSpan)
	}

	go StatsHistsRun(*statsEvery)

	log.Printf("INFO: main, listen: %s", *listen)

	log.Fatal(http.ListenAndServe(*listen, mux))
}
