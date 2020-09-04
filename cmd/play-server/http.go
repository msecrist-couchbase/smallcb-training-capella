package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

var Msgs = map[string]string{
	"session-exit": "Thanks for test-driving Couchbase!",
}

var WrongCaptchaSleepTime = 10 * time.Second

// ------------------------------------------------

func HttpMuxInit(mux *http.ServeMux) {
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(*staticDir))))

	mux.HandleFunc("/admin/stats", HttpHandleAdminStats)

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
	StatsNumInc("http.SessionExit")

	sessions.SessionExit(r.FormValue("s"))

	http.Redirect(w, r, "/?m=session-exit", http.StatusSeeOther)
}

// ------------------------------------------------

func HttpHandleSession(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("http.Session")

	data := map[string]interface{}{}

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

			sessionId, err := sessions.SessionCreate(fullName, email)
			if err == nil {
				StatsNumInc("http.Session.post.create.ok")

				http.Redirect(w, r, "/?s="+sessionId, http.StatusSeeOther)
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

	session := sessions.SessionGet(r.FormValue("s"))

	lang := r.FormValue("lang")
	code := r.FormValue("code")

	var result []byte

	ok, err := CheckLangCode(lang, code, *codeMaxLen)
	if ok {
		req := RunRequest{
			ctx:                 r.Context(),
			execUser:            ExecUser,
			execPrefix:          ExecPrefixes[lang],
			lang:                lang,
			code:                code,
			codeDuration:        *codeDuration,
			containerNamePrefix: *containerNamePrefix,
			containerVolPrefix:  *containerVolPrefix,
		}

		if session != nil {
			StatsNumInc("http.Run.session")

			result, err = RunRequestSession(
				session,
				req, readyCh,
				*containerWaitDuration, restartCh)
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
				fmt.Sprintf(", HttpHandleRun, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: HttpHandleRun, err: %v", err)
		return
	}

	StatsNumInc("http.Run.ok")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.Write(result)
}