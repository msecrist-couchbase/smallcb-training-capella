package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	// Channel of container instance #'s that are ready.
	readyCh chan int

	// Channel of container instance restart requests.
	restartCh chan Restart

	// -----------------------------------

	// The user for docker exec.
	ExecUser = "couchbase:couchbase"

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

	mux.HandleFunc("/session-exit", HttpHandleSessionExit)

	mux.HandleFunc("/session", HttpHandleSession)

	mux.HandleFunc("/run", HttpHandleRun)

	mux.HandleFunc("/", HttpHandleMain)
}

// ------------------------------------------------

func HttpHandleMain(w http.ResponseWriter, r *http.Request) {
	sessionId := r.FormValue("s")

	examplesDir := "examples"

	name := r.FormValue("name")

	// Example URL.Path == "/examples/basic-py"
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) >= 3 {
		examplesDir = parts[1] // Ex: "examples".
		name = parts[2]        // Ex: "basic-py".
	}

	lang := r.FormValue("lang")
	code := r.FormValue("code")

	MainTemplateEmit(w, *staticDir, sessionId, examplesDir, name, lang, code, nil)
}

// ------------------------------------------------

func HttpHandleSessionExit(w http.ResponseWriter, r *http.Request) {
	// TODO.

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ------------------------------------------------

func HttpHandleSession(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{}

	if r.Method == "POST" {
		errs := 0

		fullName := r.FormValue("fullName")
		if fullName == "" {
			data["errFullName"] = "full name required"
			errs += 1
		}
		data["fullName"] = fullName

		email := r.FormValue("email")
		if email == "" {
			data["errEmail"] = "email required"
			errs += 1
		}
		data["email"] = email

		if errs <= 0 {
			sessionId := "1234567" // TODO.

			http.Redirect(w, r, "/?s="+sessionId, http.StatusSeeOther)
			return
		}
	}

	template.Must(template.ParseFiles(
		*staticDir+"/session.html.template")).Execute(w, data)
}

// ------------------------------------------------

func HttpHandleRun(w http.ResponseWriter, r *http.Request) {
	lang := r.FormValue("lang")
	code := r.FormValue("code")

	var result []byte

	ok, err := CheckLangCode(lang, code, *codeMaxLen)
	if ok {
		result, err = RunLangCode(r.Context(),
			ExecUser, ExecPrefixes[lang],
			lang, code, *codeDuration, readyCh,
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
