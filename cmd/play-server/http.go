package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

var Msgs = map[string]string{
	"session-exit": "Thanks for test-driving Couchbase!",
}

var WrongCaptchaSleepTime = 5 * time.Second // To slow down robots.

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

	s := r.FormValue("s")

	session := sessions.SessionGet(s)
	if session == nil && s != "" && msg == "" {
		msg = "Session timed out."
	}

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

	MainTemplateEmit(w, *staticDir, msg, *containerPublishHost,
		session, examplesDir, name, lang, code)
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
			execUser:            ExecUser,
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

// ------------------------------------------------

func HttpProxy(listenProxy string) {
	proxyMux := http.NewServeMux()

	proxyMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		user, pswd, ok := r.BasicAuth()
		if ok {
			log.Printf("INFO: HttpProxy, path: %s, user: %s, BasicAuth",
				r.URL.Path, user)
		} else if r.URL.Path == "/uilogin" && r.Method == "POST" {
			var err error
			var saveBody io.ReadCloser

			// Duplicating r.Body since r.FormValue() consumes the r.Body.
			saveBody, r.Body, err = DupBody(r.Body)
			if err != nil {
				http.Error(w,
					http.StatusText(http.StatusInternalServerError)+
						fmt.Sprintf(", HttpProxy, DupBody, err: %v", err),
					http.StatusInternalServerError)
				log.Printf("ERROR: HttpProxy, DupBody, err: %v", err)
				return
			}

			user = r.FormValue("user")
			pswd = r.FormValue("password")

			r.Body = saveBody

			log.Printf("INFO: HttpProxy, path: %s, user: %s, uilogin",
				r.URL.Path, user)
		}

		// Default to targetPort of 10001 for serving web login UI's.
		targetPort := *containerPublishPortBase + 1

		if user != "" {
			sessionId := user + pswd

			session := sessions.SessionGet(sessionId)
			if session == nil {
				http.Error(w,
					http.StatusText(http.StatusNotFound)+
						fmt.Sprintf(", HttpProxy, session not found"),
					http.StatusNotFound)
				log.Printf("ERROR: HttpProxy, path: %s, user: %s,"+
					" session not found", r.URL.Path, user)
				return
			}

			if session.ContainerId < 0 {
				http.Error(w,
					http.StatusText(http.StatusNotFound)+
						fmt.Sprintf(", HttpProxy, session w/o container"),
					http.StatusNotFound)
				log.Printf("ERROR: HttpProxy, path: %s, user: %s,"+
					" session w/o container", r.URL.Path, user)
				return
			}

			log.Printf("INFO: HttpProxy, path: %s, user: %s, session ok,"+
				" containerId: %d, targetPort: %d", r.URL.Path, user,
				session.ContainerId, targetPort)

			// Example targetPort: 10000 + (100 * containerId) + 1 == 10001.
			targetPort = *containerPublishPortBase +
				(*containerPublishPortSpan * session.ContainerId) + 1
		}

		director := func(req *http.Request) {
			origin, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d/", targetPort))

			// req.Header.Add("X-Forwarded-Host", req.Host)
			// req.Header.Add("X-Origin-Host", origin.Host)

			req.URL.Scheme = origin.Scheme
			req.URL.Host = origin.Host
		}

		proxy := &httputil.ReverseProxy{Director: director}

		proxy.ServeHTTP(w, r)
	})

	log.Printf("INFO: HttpProxy, listenProxy: %s", listenProxy)

	log.Fatal(http.ListenAndServe(listenProxy, proxyMux))
}

// ------------------------------------------------

// DupBody is based onhttputil.DrainBody, and reads all of b to
// memory and then returns two ReadClosers yielding the same bytes.
func DupBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}

	if err = b.Close(); err != nil {
		return nil, b, err

	}

	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
