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

var (
	h = flag.Bool("h", false, "print help/usage and exit")

	help = flag.Bool("help", false, "print help/usage and exit")

	codeMaxLen = flag.Int("codeMaxLen", 16000,
		"max length of a client's request code in bytes")

	codeDuration = flag.Duration("codeDuration", 10*time.Second,
		"duration that a client's request code may run on an assigned container instance")

	containerNamePrefix = flag.String("containerNamePrefix", "smallcb-",
		"prefix of the names of container instances")

	containerVolPrefix = flag.String("containerVolPrefix", "vol-",
		"prefix of the volume directories of container instances")

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

	staticDir = flag.String("staticDir", "cmd/play-server/static",
		"path to the 'static' directory")

	listen = flag.String("listen", ":8080",
		"HTTP listen [addr]:port")

	// -----------------------------------

	readyCh chan int // Channel of container instance #'s that are ready.

	restartCh chan Restart // Channel of container instance restart requests.

	// -----------------------------------

	RunUser = "couchbase:couchbase"

	// Map from lang (or code file suffix) to execPrefix (exec command
	// prefix for executing code).
	ExecPrefixes = map[string]string{
		"java": "/run-java.sh",
	}

	// Port mapping of container port # to containerPublishPortBase + delta.
	PortMapping = [][]int{
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
)

// ------------------------------------------------

func main() {
	flag.Parse()

	if *h || *help {
		flag.Usage()
		os.Exit(2)
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

	// Have the restarters restart the required # of containers.
	for containerId := 0; containerId < *containers; containerId++ {
		restartCh <- Restart{
			ContainerId: containerId,
			DoneCh:      readyCh,
		}
	}

	mux := http.NewServeMux()

	HttpMuxInit(mux)

	log.Printf("INFO: main, listen: %s", *listen)

	log.Fatal(http.ListenAndServe(*listen, mux))
}

// ------------------------------------------------

func HttpMuxInit(mux *http.ServeMux) {
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(*staticDir))))

	mux.HandleFunc("/run", HttpHandleRun)

	mux.HandleFunc("/", HttpHandleMain)
}

// ------------------------------------------------

func HttpHandleMain(w http.ResponseWriter, r *http.Request) {
	examplesDir := "examples"

	name := r.FormValue("name")

	// Example URL.Path == "/examples/basic-py"
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) >= 3 {
		examplesDir = parts[1] // Ex: "examples".
		name = parts[2]        // Ex: "basic-py".
	}

	if strings.HasPrefix(r.URL.Path, "/examples/") &&
		len(r.URL.Path) > len("/examples/") {
		name = r.URL.Path[len("/examples/"):]
	}

	lang := r.FormValue("lang")
	code := r.FormValue("code")

	MainTemplateEmit(w, *staticDir, examplesDir, name, lang, code)
}

// ------------------------------------------------

func HttpHandleRun(w http.ResponseWriter, r *http.Request) {
	lang := r.FormValue("lang")
	code := r.FormValue("code")

	var result []byte

	ok, err := CheckLangCode(lang, code, *codeMaxLen)
	if ok {
		result, err = RunLangCode(r.Context(), RunUser,
			ExecPrefixes[lang], lang, code, *codeDuration, readyCh,
			*containerWaitDuration,
			*containerNamePrefix,
			*containerVolPrefix,
			restartCh)
	}

	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", HttpHandleRun, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: HttpHandleRun, err: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.Write(result)
}
