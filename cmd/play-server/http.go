package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var Msgs = map[string]string{
	"session-exit":    "Thanks for test-driving Couchbase!",
	"session-timeout": "Session timed out.",
}

var WrongCaptchaSleepTime = 5 * time.Second // To slow down robots.

// ------------------------------------------------

func HttpMuxInit(mux *http.ServeMux) {
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(*staticDir))))

	mux.HandleFunc("/admin/dashboard", HttpHandleAdminDashboard)

	mux.HandleFunc("/admin/stats", HttpHandleAdminStats) // Returns JSON.

	mux.HandleFunc("/admin/sessions-release-containers",
		HttpHandleAdminSessionsReleaseContainers)

	mux.HandleFunc("/session-exit", HttpHandleSessionExit)

	mux.HandleFunc("/session", HttpHandleSession)

	mux.HandleFunc("/run", HttpHandleRun)

	mux.HandleFunc("/", HttpHandleMain)
}

// ------------------------------------------------

func HttpHandleMain(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("http.Main")

	msg := r.FormValue("m")
	if Msgs[msg] != "" {
		msg = Msgs[msg]
	}

	s := r.FormValue("s")

	session := sessions.SessionGet(s)
	if session == nil && s != "" && msg == "" {
		http.Redirect(w, r, "/?m=session-timeout", http.StatusSeeOther)

		return
	}

	examplesPath := "examples"

	name := r.FormValue("name")

	// Example URL.Path == "/examples/basic-py"
	path := r.URL.Path

	if strings.Index(path, "..") >= 0 {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		log.Printf("ERROR: HttpHandleMain, err: path has '..'")
		return
	}

	for len(path) > 0 && strings.HasSuffix(path, "/") {
		path = path[0 : len(path)-1]
	}

	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		// Ex: "examples" or "examples-more/foo/bar".
		examplesPath = strings.Join(parts[1:len(parts)-1], "/")

		// Ex: "basic-py".
		name = parts[len(parts)-1]
	}

	lang := r.FormValue("lang")
	code := r.FormValue("code")

	portApp, _ := strconv.Atoi(strings.Split(*listen, ":")[1])

	MainTemplateEmit(w, *staticDir, msg, *host, portApp, *version,
		session, *sessionsMaxAge, examplesPath, name, lang, code)
}

// ------------------------------------------------

func HttpHandleSessionExit(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("http.SessionExit")

	sessions.SessionExit(r.FormValue("s"))

	http.Redirect(w, r, "/?m=session-exit", http.StatusSeeOther)
}

// ------------------------------------------------

func HttpHandleSession(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("http.Session")

	data := map[string]interface{}{
		"sessionsMaxAge": strings.Replace(
			sessionsMaxAge.String(), "m0s", " min", 1),
	}

	if r.Method == "POST" {
		StatsNumInc("http.Session.post")

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
			StatsNumInc("http.Session.post.err.captcha")

			time.Sleep(WrongCaptchaSleepTime)

			data["errCaptcha"] = "please guess again"
			errs += 1
		}

		if errs <= 0 {
			StatsNumInc("http.Session.post.create")

			session, err := sessions.SessionCreate(fullName, email)
			if err == nil && session != nil && session.SessionId != "" {
				StatsNumInc("http.Session.post.create.ok")

				req := RunRequest{
					ctx:                 context.Background(),
					execPrefix:          "",
					lang:                "n/a",
					code:                "n/a",
					codeDuration:        *codeDuration,
					containerNamePrefix: *containerNamePrefix,
					containerVolPrefix:  *containerVolPrefix,
					cbAdminPassword:     CBAdminPassword,
				}

				// Async attempt to assign a container instance to
				// the new session, so the client doesn't wait.
				go SessionAssignContainer(session, req,
					readyCh, *containerWaitDuration, restartCh,
					*containers, *containersSingleUse)

				http.Redirect(w, r, "/?s="+session.SessionId,
					http.StatusSeeOther)

				return
			}

			StatsNumInc("http.Session.post.create.err")

			data["err"] = fmt.Sprintf("Could not create session - "+
				"please try again later. (%v)", err)
		}

		StatsNumInc("http.Session.post.err")
	} else {
		StatsNumInc("http.Session.get")
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
	StatsNumInc("http.Run")

	s := r.FormValue("s")

	session := sessions.SessionGet(s)
	if session == nil && s != "" {
		http.Error(w,
			http.StatusText(http.StatusNotFound)+
				", session unknown",
			http.StatusNotFound)
		log.Printf("ERROR: HttpHandleRun, session unknown, s: %v", s)
		return
	}

	lang := r.FormValue("lang")
	code := r.FormValue("code")

	var result []byte

	ok, err := CheckLangCode(lang, code, *codeMaxLen)
	if ok {
		req := RunRequest{
			ctx:                 r.Context(),
			execPrefix:          ExecPrefixes[lang],
			lang:                lang,
			code:                code,
			codeDuration:        *codeDuration,
			containerNamePrefix: *containerNamePrefix,
			containerVolPrefix:  *containerVolPrefix,
			cbAdminPassword:     CBAdminPassword,
		}

		if session != nil {
			StatsNumInc("http.Run.session")

			result, err = RunRequestSession(
				session, req, readyCh,
				*containerWaitDuration, restartCh,
				*containers, *containersSingleUse)
			if err != nil {
				StatsNumInc("http.Run.session.err")
			} else {
				StatsNumInc("http.Run.session.ok")
			}
		} else {
			StatsNumInc("http.Run.single")

			result, err = RunRequestSingle(
				req, readyCh,
				*containerWaitDuration, restartCh)
			if err != nil {
				StatsNumInc("http.Run.single.err")
			} else {
				StatsNumInc("http.Run.single.ok")
			}
		}
	}

	if err != nil {
		StatsNumInc("http.Run.err")

		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", HttpHandleRun, err: %v\n"+
					"------------------------\n%s\n",
					err, result),
			http.StatusInternalServerError)
		log.Printf("ERROR: HttpHandleRun, err: %v", err)
		return
	}

	StatsNumInc("http.Run.ok")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.Write(result)
}
