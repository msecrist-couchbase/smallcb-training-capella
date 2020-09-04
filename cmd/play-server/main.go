package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
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

	// -----------------------------------

	Msgs = map[string]string{
		"session-exit": "Thanks for test-driving Couchbase!",
	}
)

// ------------------------------------------------

func main() {
	StatsInfo("main.startTime", time.Now().Format("2006-01-02T15:04:05.000-07:00"))

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

	mux.HandleFunc("/admin/stats", HttpHandleAdminStats)

	mux.HandleFunc("/session-exit", HttpHandleSessionExit)

	mux.HandleFunc("/session", HttpHandleSession)

	mux.HandleFunc("/run", HttpHandleRun)

	mux.HandleFunc("/", HttpHandleMain)
}

// ------------------------------------------------

func HttpHandleMain(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("tot.http.main")

	msg := r.FormValue("m")
	if Msgs[msg] != "" {
		msg = Msgs[msg]
	}

	session := sessions.SessionGet(r.FormValue("s"))

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

	MainTemplateEmit(w, *staticDir, msg, session,
		examplesDir, name, lang, code)
}

// ------------------------------------------------

func HttpHandleSessionExit(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("tot.http.session-exit")

	sessions.SessionExit(r.FormValue("s"))

	http.Redirect(w, r, "/?m=session-exit", http.StatusSeeOther)
}

// ------------------------------------------------

func HttpHandleSession(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("tot.http.session")

	data := map[string]interface{}{}

	if r.Method == "POST" {
		StatsNumInc("tot.http.session.post")

		errs := 0

		fullName := strings.TrimSpace(r.FormValue("fullName"))
		if fullName == "" {
			data["errFullName"] = "full name required"
			errs += 1
		}
		data["fullName"] = fullName

		email := strings.TrimSpace(r.FormValue("email"))
		if email == "" {
			data["errEmail"] = "email required"
			errs += 1
		}
		data["email"] = email

		captcha := strings.TrimSpace(r.FormValue("captcha"))
		if captcha == "" {
			data["errCaptcha"] = "guess required"
			errs += 1
		} else if !CaptchaCheck(captcha) {
			StatsNumInc("tot.http.session.post.err-captcha")

			time.Sleep(10 * time.Second)

			data["errCaptcha"] = "please guess again"
			errs += 1
		}

		if errs <= 0 {
			StatsNumInc("tot.http.session.post.create")

			sessionId, err := sessions.SessionCreate(fullName, email)
			if err == nil {
				StatsNumInc("tot.http.session.post.create.ok")

				http.Redirect(w, r, "/?s="+sessionId, http.StatusSeeOther)
				return
			}

			StatsNumInc("tot.http.session.post.create.err")

			data["err"] = fmt.Sprintf("Could not create session - "+
				"please try again later. (%v)", err)
		}

		StatsNumInc("tot.http.session.post.err")
	} else {
		StatsNumInc("tot.http.session.get")
	}

	captchaURL, err := CaptchaGenerateBase64ImageDataURL(240, 80, *maxCaptchas)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", CaptchaGenerate, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: CaptchaGenerate, err: %v", err)
		return
	}

	data["captchaSrc"] = template.HTMLAttr("src=\"" + captchaURL + "\"")

	template.Must(template.ParseFiles(
		*staticDir+"/session.html.template")).Execute(w, data)
}

// ------------------------------------------------

func HttpHandleRun(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("tot.http.run")

	session := sessions.SessionGet(r.FormValue("s"))

	lang := r.FormValue("lang")
	code := r.FormValue("code")

	var result []byte

	ok, err := CheckLangCode(lang, code, *codeMaxLen)
	if ok {
		if session != nil {
			StatsNumInc("tot.http.run.session")

			result, err = RunLangCodeSession(
				r.Context(), session,
				ExecUser, ExecPrefixes[lang],
				lang, code, *codeDuration, readyCh,
				*containerWaitDuration,
				*containerNamePrefix,
				*containerVolPrefix,
				restartCh)
			if err != nil {
				StatsNumInc("tot.http.run.session.err")
			} else {
				StatsNumInc("tot.http.run.session.ok")
			}
		} else {
			StatsNumInc("tot.http.run.single")

			result, err = RunLangCode(
				r.Context(),
				ExecUser, ExecPrefixes[lang],
				lang, code, *codeDuration, readyCh,
				*containerWaitDuration,
				*containerNamePrefix,
				*containerVolPrefix,
				restartCh)
			if err != nil {
				StatsNumInc("tot.http.run.single.err")
			} else {
				StatsNumInc("tot.http.run.single.ok")
			}
		}
	}

	if err != nil {
		StatsNumInc("tot.http.run.err")

		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", HttpHandleRun, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: HttpHandleRun, err: %v", err)
		return
	}

	StatsNumInc("tot.http.run.ok")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.Write(result)
}
