package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/semver"
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

	mux.HandleFunc("/static-data", HttpHandleStaticData)

	mux.HandleFunc("/admin/health", HttpHandleHealth)

	mux.HandleFunc("/admin/dashboard", HttpHandleAdminDashboard)

	mux.HandleFunc("/admin/stats", HttpHandleAdminStats) // Returns JSON.

	mux.HandleFunc("/admin/sessions-release-containers",
		HttpHandleAdminSessionsReleaseContainers)

	mux.HandleFunc("/session-exit", HttpHandleSessionExit)

	mux.HandleFunc("/session-info", HttpHandleSessionInfo)

	mux.HandleFunc("/session", HttpHandleSession)

	mux.HandleFunc("/target", HttpHandleTarget)

	mux.HandleFunc("/target-exit", HttpHandleTargetExit)

	mux.HandleFunc("/run", HttpHandleRun)

	mux.HandleFunc("/et", HttpHandleET)

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
		StatsNumInc("http.Main.err", "http.Main.err.session-timeout")

		http.Redirect(w, r, "/?m=session-timeout",
			http.StatusSeeOther)

		return
	}

	targetCookie, terr := r.Cookie(*targetsCookieName)
	Target := target{}
	if terr == nil && targetCookie != nil {
		targetCookieValue := DecryptText(targetCookie.Value)
		targetTuple := strings.Split(targetCookieValue, "::")
		Target.DBurl = targetTuple[0]
		Target.DBuser = targetTuple[1]
		Target.DBpwd = targetTuple[2]
		Target.DBHost = GetDBHostFromURL(Target.DBurl)
		if len(targetTuple) >= 4 {
			Target.ExpiryTime = targetTuple[3]
		} else {
			Target.ExpiryTime = ""
		}
		if runtime.GOOS == "linux" {
			addSrvRoute(Target.DBurl)
		} else {
			*natPublicIP = "YourHostIP"
		}
		Target.NatPublicIP = *natPublicIP
		Target.NetworkStatus, Target.Version, Target.IPv4 = CheckDBAccess(Target.DBurl)
		Target.Status = Target.NetworkStatus
		if Target.NetworkStatus == "OK" {
			Target.UserAccessStatus = CheckDBUserAccess(Target.IPv4, Target.DBuser, Target.DBpwd)
			Target.Status = Target.UserAccessStatus
			if Target.UserAccessStatus == "OK" {
				Target.SampleAccessStatus = CheckDBUserSampleAccess(Target.IPv4, Target.DBuser, Target.DBpwd, "travel-sample")
				Target.Status = Target.SampleAccessStatus
			}
		}
	}

	examplesPath := "examples"

	name := r.FormValue("name")

	// Example URL.Path == "/examples/basic-py"
	path := r.URL.Path

	if strings.Index(path, "..") >= 0 {
		StatsNumInc("http.Main.err", "http.Main.err.path")

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
	view := r.FormValue("view")

	portApp, _ := strconv.Atoi(strings.Split(*listen, ":")[1])

	bodyClass := r.FormValue("bodyClass")
	if bodyClass == "" {
		bodyClass = "dark"
	}

	program := r.FormValue("program")

	code = CodeFromFixup(code, program, lang, r.FormValue("from"))

	err := CheckVerSDK(lang, r.FormValue("verSDK"))
	if err != nil {
		StatsNumInc("http.Main.err", "http.Main.err.checkVerSDK")

		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)

		log.Printf("ERROR: HttpHandleMain, err: %v", err)

		return
	}

	err = MainTemplateEmit(w, *staticDir, msg, *host, portApp,
		*version, VersionSDKs, session, *sessionsMaxAge, *sessionsMaxIdle,
		Target,
		*listenPortBase, *listenPortSpan, PortMapping,
		examplesPath, name, r.FormValue("title"),
		lang, code, r.FormValue("highlight"), view, bodyClass,
		r.FormValue("infoBefore"),
		r.FormValue("infoAfter"))
	if err != nil {
		StatsNumInc("http.Main.err", "http.Main.err.template")

		return
	}

	StatsNumInc("http.Main.ok")
}

func EncryptText(cleartext string) (outputText string) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Error while doing encrypt!")
			outputText = cleartext
		}
	}()
	return Encrypt(*encryptKey, cleartext)
}

func DecryptText(ettext string) (outputText string) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Error while doing decrypt!")
			outputText = ettext
		}
	}()
	return Decrypt(*encryptKey, ettext)
}

// ------------------------------------------------

func HttpHandleSessionExit(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("http.SessionExit")

	sessions.SessionExit(r.FormValue("s"))

	url := r.FormValue("ebase")
	if url == "" {
		url = "/"
	}

	e := r.FormValue("e")
	if e != "" {
		url = url + e
	}

	if strings.Index(url, "?") < 0 {
		url += "?m=session-exit"
	} else {
		url += "&m=session-exit"
	}

	http.Redirect(w, r, url, http.StatusSeeOther)
}

func HttpHandleTargetExit(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("http.TargetExit")

	//delete route before exit
	targetCookie, terr := r.Cookie(*targetsCookieName)
	Target := target{}
	if terr == nil && targetCookie != nil {
		targetCookieValue := DecryptText(targetCookie.Value)
		targetTuple := strings.Split(targetCookieValue, "::")
		Target.DBurl = targetTuple[0]
		if runtime.GOOS == "linux" {
			delSrvRoute(Target.DBurl)
		} else {
			*natPublicIP = "YourHostIP"
			Target.NatPublicIP = *natPublicIP
		}
	}
	//set cookie expire time to -1
	targetsCookie := &http.Cookie{
		Name:   *targetsCookieName,
		MaxAge: -1,
	}
	http.SetCookie(w, targetsCookie)

	url := r.FormValue("ebase")
	if url == "" {
		url = "/"
	}

	e := r.FormValue("e")
	if e != "" {
		url = url + e
	}

	http.Redirect(w, r, url, http.StatusSeeOther)
}

// ------------------------------------------------

func HttpHandleSessionInfo(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("http.SessionInfo")

	d, err := sessions.SessionInfo(r.FormValue("s"))
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)

		log.Printf("ERROR: HttpHandleSessionInfo, err: %v", err)

		return
	}

	j, err := json.Marshal(d)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)

		log.Printf("ERROR: HttpHandleSessionInfo, err: %v", err)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(j)
}

// ------------------------------------------------

var regexpE = regexp.MustCompile(`^[a-zA-Z0-9_#=/\-\?\.]*$`)

func HttpHandleSession(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("http.Session")

	if *host != "127.0.0.1" && *host != "localhost" &&
		strings.Split(r.Host, ":")[0] != *host {
		StatsNumInc("http.Session.redirect.host")

		var suffix string

		if r.ParseForm() == nil {
			suffix = r.Form.Encode()
			if len(suffix) > 0 {
				suffix = "?" + suffix
			}
		}

		http.Redirect(w, r, "http://"+*host+"/session"+suffix, http.StatusSeeOther)

		log.Printf("INFO: Session redirect, from host: %v, to host: %s, suffix: %s", r.Host, *host, suffix)

		return
	}

	// Optional extra URL suffix for redirect on success,
	// used to target a particular code example or tour.
	e := r.FormValue("e")
	if !regexpE.MatchString(e) || strings.Index(e, "..") >= 0 {
		StatsNumInc("http.Session.err", "http.Session.err.bad-e")

		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)

		log.Printf("ERROR: HttpHandleMain, e: %s, err: e unmatched", e)

		return
	}

	bodyClass := r.FormValue("bodyClass")
	if bodyClass == "" {
		bodyClass = "dark"
	}

	data := map[string]interface{}{
		"AnalyticsHTML": template.HTML(AnalyticsHTML(*host)),
		"OptanonHTML":   template.HTML(OptanonHTML(*host)),
		"SessionsMaxAge": strings.Replace(
			sessionsMaxAge.String(), "m0s", " min", 1),
		"SessionsMaxIdle": strings.Replace(
			sessionsMaxIdle.String(), "m0s", " min", 1),
		"title":         r.FormValue("title"),
		"intro":         r.FormValue("intro"),
		"namec":         r.FormValue("namec"),
		"emailc":        r.FormValue("emailc"),
		"captchac":      r.FormValue("captchac"),
		"defaultBucket": r.FormValue("defaultBucket"),
		"groupSize":     r.FormValue("groupSize"),
		"init":          r.FormValue("init"),
		"e":             e,
		"bodyClass":     bodyClass,
	}

	if r.Method == "POST" {
		StatsNumInc("http.Session.post")

		gen := fmt.Sprintf("gen!%s-%d",
			time.Now().Format("2006/01/02-15:04:05"),
			rand.Intn(1000000))

		errs := 0

		name := strings.TrimSpace(r.FormValue("name"))
		if r.FormValue("namec") == "skip" {
			name = gen
		}
		if name == "" {
			StatsNumInc("http.Session.post.err.name")

			data["errName"] = "name required"
			errs += 1
		}
		data["name"] = name

		email := strings.TrimSpace(r.FormValue("email"))
		if r.FormValue("emailc") == "skip" {
			email = gen
		}
		if email == "" {
			StatsNumInc("http.Session.post.err.email")

			data["errEmail"] = "email required"
			errs += 1
		}
		data["email"] = email

		captcha := strings.TrimSpace(r.FormValue("captcha"))
		if r.FormValue("captchac") != "skip" {
			if captcha == "" {
				data["errCaptcha"] = "guess required"
				errs += 1
			} else if !CaptchaCheck(captcha) {
				StatsNumInc("http.Session.post.err.captcha")

				time.Sleep(WrongCaptchaSleepTime)

				data["errCaptcha"] = "please guess again"
				errs += 1
			}
		}

		groupSize, err := strconv.Atoi(strings.TrimSpace(r.FormValue("groupSize")))
		if err != nil || groupSize < 1 {
			groupSize = 1
		}

		// Racy-y check to see if there's enough containers available,
		// where it's cheaper to check this early rather than trying
		// to allocate containers and fail partway through.
		_, sessionsCountWithContainer := sessions.Count()
		if *containers-int(sessionsCountWithContainer)-groupSize < *containersSingleUse {
			data["err"] = fmt.Sprintf("Not enough resources right now - " +
				"please try again later.")
			errs += 1
		}

		if errs <= 0 {
			StatsNumInc("http.Session.post.create")

			session, err := sessions.SessionCreate("", name, email)
			if err == nil && session != nil && session.SessionId != "" {
				StatsNumInc("http.Session.post.ok", "http.Session.post.create.assign")

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

				defaultBucket := r.FormValue("defaultBucket")
				if defaultBucket == "" {
					defaultBucket = "travel-sample"
				}

				_, err = SessionAssignContainer(
					session, req, readyCh,
					*containerWaitDuration, restartCh,
					*containers, *containersSingleUse,
					r.FormValue("init"), "0",
					defaultBucket)

				for i := 1; err == nil && i < groupSize; i++ {
					var childSession *Session

					childSession, err = sessions.SessionCreate(
						session.SessionId,
						fmt.Sprintf("~%s-%d", session.SessionId, i),
						"~")
					if err != nil {
						break
					}

					_, err = SessionAssignContainer(
						childSession, req, readyCh,
						*containerWaitDuration, restartCh,
						*containers, *containersSingleUse,
						r.FormValue("init"), fmt.Sprintf("%d", i),
						defaultBucket)
				}

				if err == nil {
					StatsNumInc("http.Session.post.ok", "http.Session.post.create.assign.ok")

					url := r.FormValue("ebase")
					if url == "" {
						url = "/"
					}

					if e != "" {
						url = url + e
					}

					if strings.Index(url, "?") < 0 {
						url += "?s=" + session.SessionId
					} else {
						url += "&s=" + session.SessionId
					}

					http.Redirect(w, r, url, http.StatusSeeOther)

					StatsNumInc("http.Session.post.ok", "http.Session.post.create.ok")

					return
				}

				StatsNumInc("http.Session.post.ok", "http.Session.post.create.assign.err")

				sessions.SessionExit(session.SessionId)
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
		*staticDir+"/session.html.tmpl")).Execute(w, data)
}

func HttpHandleTarget(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("http.Target")

	if *host != "127.0.0.1" && *host != "localhost" &&
		strings.Split(r.Host, ":")[0] != *host {
		StatsNumInc("http.Target.redirect.host")

		var suffix string

		if r.ParseForm() == nil {
			suffix = r.Form.Encode()
			if len(suffix) > 0 {
				suffix = "?" + suffix
			}
		}

		http.Redirect(w, r, "http://"+*host+"/target"+suffix, http.StatusSeeOther)

		log.Printf("INFO: Target redirect, from host: %v, to host: %s, suffix: %s", r.Host, *host, suffix)

		return
	}

	// Optional extra URL suffix for redirect on success,
	// used to target a particular code example or tour.
	e := r.FormValue("e")
	if !regexpE.MatchString(e) || strings.Index(e, "..") >= 0 {
		StatsNumInc("http.Session.err", "http.Session.err.bad-e")

		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)

		log.Printf("ERROR: HttpHandleMain, e: %s, err: e unmatched", e)

		return
	}

	bodyClass := r.FormValue("bodyClass")
	if bodyClass == "" {
		bodyClass = "dark"
	}

	targetCookie, terr := r.Cookie(*targetsCookieName)
	Target := target{}
	if terr == nil && targetCookie != nil {
		targetCookieValue := DecryptText(targetCookie.Value)
		targetTuple := strings.Split(targetCookieValue, "::")
		Target.DBurl = targetTuple[0]
		Target.DBuser = targetTuple[1]
		Target.DBpwd = targetTuple[2]
		Target.DBHost = GetDBHostFromURL(Target.DBurl)
		if len(targetTuple) >= 4 {
			Target.ExpiryTime = targetTuple[3]
		} else {
			Target.ExpiryTime = ""
		}
		if runtime.GOOS == "linux" {
			addSrvRoute(Target.DBurl)
		} else {
			*natPublicIP = "YourHostIP"
		}
		Target.NatPublicIP = *natPublicIP
		Target.NetworkStatus, Target.Version, Target.IPv4 = CheckDBAccess(Target.DBurl)
		Target.Status = Target.NetworkStatus
		if Target.NetworkStatus == "OK" {
			Target.UserAccessStatus = CheckDBUserAccess(Target.IPv4, Target.DBuser, Target.DBpwd)
			Target.Status = Target.UserAccessStatus
			if Target.UserAccessStatus == "OK" {
				Target.SampleAccessStatus = CheckDBUserSampleAccess(Target.IPv4, Target.DBuser, Target.DBpwd, "travel-sample")
				Target.Status = Target.SampleAccessStatus
			}
		}

	}
	data := map[string]interface{}{
		"AnalyticsHTML": template.HTML(AnalyticsHTML(*host)),
		"OptanonHTML":   template.HTML(OptanonHTML(*host)),
		"TargetsMaxAge": ((*targetsMaxAge).Hours() / 24),
		"title":         r.FormValue("title"),
		"intro":         r.FormValue("intro"),
		"dburlc":        r.FormValue("dburlc"),
		"dbuserc":       r.FormValue("dbuserc"),
		"dbpwdc":        r.FormValue("dbpwdc"),
		"dburl":         r.FormValue("dburl"),
		"dbuser":        r.FormValue("dbuser"),
		"dbpwd":         r.FormValue("dbpwd"),
		"natpublicip":   *natPublicIP,
		"bodyClass":     bodyClass,
	}

	if runtime.GOOS != "linux" {
		*natPublicIP = "YourHostIP"
		data["natpublicip"] = *natPublicIP
	}

	if r.Method == "POST" {
		StatsNumInc("http.Target.post")

		gen := fmt.Sprintf("gen!%s-%d",
			time.Now().Format("2006/01/02-15:04:05"),
			rand.Intn(1000000))

		errs := 0

		dburl := strings.TrimSpace(r.FormValue("dburl"))
		if r.FormValue("dburlc") == "skip" {
			dburl = gen
		}
		if dburl == "" {
			StatsNumInc("http.Target.post.err.dburl")

			data["errDBurl"] = "db URL required"
			errs += 1
		} else {
			_, _, err := net.LookupSRV("couchbases", "tcp", dburl)
			if err != nil {
				data["errDBurl"] = "db URL invalid or not reachable"
				errs += 1
			}
		}

		dbHost := dburl
		if !strings.HasPrefix(dburl, "couchbase:") && !strings.HasPrefix(dburl, "couchbases:") && strings.Contains(dburl, "cloud.couchbase.com") {
			dburl = "couchbases://" + dburl
		}
		if strings.HasPrefix(dburl, "couchbases:") && !strings.Contains(dburl, "ssl=no_verify") {
			if !strings.Contains(dburl, "?") {
				dburl += "?ssl=no_verify"
			} else {
				dburl += "&ssl=no_verify"
			}
		}

		data["dburl"] = dburl

		dbuser := strings.TrimSpace(r.FormValue("dbuser"))
		if r.FormValue("dbuserc") == "skip" {
			dbuser = gen
		}
		if dbuser == "" {
			StatsNumInc("http.Target.post.err.dbuser")

			data["errDBuser"] = "db user required"
			errs += 1
		}
		data["dbuser"] = dbuser

		dbpwd := strings.TrimSpace(r.FormValue("dbpwd"))
		if r.FormValue("dbpwdc") == "skip" {
			dbpwd = gen
		}
		if dbpwd == "" {
			StatsNumInc("http.Target.post.err.dbpwd")

			data["errDBpwd"] = "db user password required"
			errs += 1
		}
		data["dbpwd"] = dbpwd

		//fmt.Printf("INFO: Target POST, dburl: %s, dbuser: %s, dbpwd: %s, errs: %v\n", dburl, dbuser, dbpwd, errs)
		if errs <= 0 {
			// Encrypt and Set cookie
			//fmt.Println(data)
			time.FixedZone("PDT", 8*60*60)
			age := time.Now().Add(*targetsMaxAge)
			cookieValue := dburl + "::" + dbuser + "::" + dbpwd + "::" + age.Format(time.RFC3339Nano)
			etCookieValue := EncryptText(cookieValue)
			//fmt.Println("etCookieValue=" + etCookieValue)
			targetsCookie := &http.Cookie{
				Name:   *targetsCookieName,
				Value:  etCookieValue,
				MaxAge: int((*targetsMaxAge).Seconds()),
			}
			if runtime.GOOS == "linux" {
				addSrvRoute(dbHost)
			} else {
				*natPublicIP = "YourHostIP"
				data["natpublicip"] = *natPublicIP
			}
			Target.NatPublicIP = *natPublicIP
			http.SetCookie(w, targetsCookie)
			url := r.FormValue("ebase")
			if url == "" {
				url = "/"
			}

			if e != "" {
				url = url + e
			}

			http.Redirect(w, r, url, http.StatusSeeOther)
			StatsNumInc("http.Target.post.ok", "http.Target.post.create.ok")
			return
		}
		StatsNumInc("http.Target.post.err")

	} else {
		StatsNumInc("http.Target.get")
	}

	template.Must(template.ParseFiles(
		*staticDir+"/target.html.tmpl")).Execute(w, data)
}

// ------------------------------------------------

// Executes some code posted in request body
// parameters:
// - s: session
// - lang: language of the code
// - code: code to run
// - color: ???????
func HttpHandleRun(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("http.Run")

	s := r.FormValue("s")

	color := r.FormValue("color")

	session := sessions.SessionGet(s)
	if session == nil && s != "" {
		// if session key was passed by there was no session with such key
		StatsNumInc("http.Run.err")

		t := http.StatusText(http.StatusNotFound) +
			", err: session unknown"

		EmitOutput(w, t, color)

		log.Printf("ERROR: HttpHandleRun, session unknown, s: %v", s)

		return
	}

	lang := r.FormValue("lang")

	// Concatenate all form values whose name starts with
	// a "code" prefix, sorted by the name.

	var codeKeys []string
	for k := range r.Form {
		if strings.HasPrefix(k, "code") {
			codeKeys = append(codeKeys, k)
		}
	}

	sort.Strings(codeKeys)

	var codeVals []string
	for _, k := range codeKeys {
		codeVals = append(codeVals, r.Form[k]...)
	}

	code := strings.Join(codeVals, "")
	from := r.FormValue("from")

	code = CodeFromFixup(code, r.FormValue("program"), lang, from)

	err := CheckVerSDK(lang, r.FormValue("verSDK"))
	if err != nil {
		StatsNumInc("http.Run.err", "http.Run.err.checkVerSDK")

		t := http.StatusText(http.StatusBadRequest) +
			fmt.Sprintf("Oops, SDK version %s isn't supported here.\n"+
				"------------------------\n"+
				"HttpHandleRun, err: %v",
				r.FormValue("verSDK"), err)

		EmitOutput(w, t, color)

		return
	}

	err = CheckVerServer(r.FormValue("verServer"))
	if err != nil {
		StatsNumInc("http.Run.err", "http.Run.err.checkVerServer")

		t := http.StatusText(http.StatusBadRequest) +
			fmt.Sprintf("Oops, server version %s isn't supported here.\n"+
				"------------------------\n"+
				"HttpHandleRun, err: %v",
				r.FormValue("verServer"), err)

		EmitOutput(w, t, color)

		return
	}

	var result []byte

	var req RunRequest

	ok, err := CheckLangCode(lang, code, *codeMaxLen)
	if ok {
		req = RunRequest{
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
	var t string
	if err != nil {
		StatsNumInc("http.Run.err")
		// If there is a run time error, hide it from the Documentation runs. Continue to log the error on the server.
		if from == "docs" {
			t = "Sorry, our servers are in maintenance.\nThese servers will be back soon.\n"
		} else {
			t = http.StatusText(http.StatusInternalServerError) +
				fmt.Sprintf(", HttpHandleRun, err: %v\n"+
					"------------------------\n%s\n",
					err, result)
		}
		if strings.Index(t, "err: timeout") > 0 {
			t = "Whoops, timeout error.\n" +
				" -- perhaps try again later\n" +
				"    as the server might be overloaded." +
				"\n\n\n\n" + t
		}

		// Avoid actual error and use 200 OK so that output
		// appears correctly.
		//
		// http.Error(w, t, http.StatusInternalServerError)

		EmitOutput(w, t, color)

		req.cbAdminPassword = "" // To avoid log.

		err = fmt.Errorf("HttpHandleRun, req: %+v, err: %v", req, err)

		log.Printf("ERROR: HttpHandleRun, err: %v", err)

		StatsErr(err)

		return
	}

	StatsNumInc("http.Run.ok")

	EmitOutput(w, string(result), color)
}

func HttpHandleStaticData(w http.ResponseWriter, r *http.Request) {
	path := r.FormValue("path")

	if strings.Index(path, "..") >= 0 {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)

		log.Printf("ERROR: HttpHandleStaticData, err: path has '..'")

		return
	}

	m, err := ReadYaml(*staticDir + "/" + path + ".yaml")
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)

		log.Printf("ERROR: HttpHandleStaticData, err: %v", err)

		return
	}

	d := CleanupInterfaceValue(m)

	j, err := json.Marshal(d)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)

		log.Printf("ERROR: HttpHandleStaticData, err: %v", err)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(j)
}

// ------------------------------------------------

func EmitOutput(w http.ResponseWriter, result, color string) {
	data := map[string]interface{}{
		"AnalyticsHTML": template.HTML(AnalyticsHTML(*host)),
		"Output":        string(result),
		"Color":         color,
	}

	w.Header().Set("Content-Type", "text/html")

	template.Must(template.ParseFiles(
		*staticDir+"/output.html.tmpl")).Execute(w, data)
}

// ------------------------------------------------

func AnalyticsHTML(host string) string {
	if host == "127.0.0.1" || host == "localhost" ||
		strings.Index(*jsFlags, "allOff") >= 0 ||
		strings.Index(*jsFlags, "analyticsOff") >= 0 {
		return ""
	}

	return `<script>(function(w,d,s,l,i){w[l]=w[l]||[];w[l].push({'gtm.start':
new Date().getTime(),event:'gtm.js'});var f=d.getElementsByTagName(s)[0],
j=d.createElement(s),dl=l!='dataLayer'?'&l='+l:'';j.async=true;j.src=
'https://www.googletagmanager.com/gtm.js?id='+i+dl;f.parentNode.insertBefore(j,f);
})(window,document,'script','dataLayer','GTM-MVPNN2');</script>`
}

// ------------------------------------------------

func OptanonHTML(host string) string {
	if host == "127.0.0.1" || host == "localhost" ||
		strings.Index(*jsFlags, "allOff") >= 0 ||
		strings.Index(*jsFlags, "optanonOff") >= 0 {
		return ""
	}

	return `<link type="text/css" rel="stylesheet" href="https://cdn.cookielaw.org/skins/4.3.3/default_flat_bottom_two_button_black/v2/css/optanon.css"/>
<script src="https://cdn.cookielaw.org/scripttemplates/otSDKStub.js" type="text/javascript" charset="UTF-8" data-domain-script="589e23c3-a7c6-4ff3-a948-b7d86b33b846"></script>
<script>function OptanonWrapper(){}</script>`
}

// ------------------------------------------------
// Add the session info to the navigation URLs in code samples
// Replace the end of the URL with ?s=session_id
func AddSessionInfo(session *Session, infoBefore string) string {
	if session != nil {
		sIDNext := fmt.Sprintf("?s=%s' class=\"next-button\"", session.SessionId)
		sIDPrev := fmt.Sprintf("?s=%s' class=\"previous-button\"", session.SessionId)
		infoBefore = strings.ReplaceAll(infoBefore, "' class=\"next-button\"", sIDNext)
		infoBefore = strings.ReplaceAll(infoBefore, "' class=\"prev-button\"", sIDPrev)
	}
	return infoBefore
}

// ------------------------------------------------

func CodeFromFixup(code, program, lang, from string) string {
	if from == "docs" {
		code = strings.ReplaceAll(code, "\"Administrator\"", "\"username\"")
		code = strings.ReplaceAll(code, "'Administrator'", "'username'")

		if lang == "java" {
			code = strings.ReplaceAll(code,
				"public class "+program, "class Program")
		}

		if lang == "scala" {
			code = strings.ReplaceAll(code,
				"object "+program, "object Program")
		}

		if lang == "dotnet" {
			code = strings.ReplaceAll(code,
				"class "+program, "class Program")
		}
	}
	// replace if encrypted text used %%%<ET>%%%
	for strings.Contains(code, "%%%") {
		re := regexp.MustCompile("%%%(.*?)%%%")
		match := re.FindStringSubmatch(code)
		et := strings.Split(match[0], "%%%")
		ct := Decrypt(*encryptKey, et[1])
		code = strings.ReplaceAll(code, "%%%"+match[1]+"%%%", ct)
	}

	return code
}

// ------------------------------------------------

// Returns true if the wanted SDK version is compatible with
// the available SDK version.
func CheckVerSDK(langIn, verSDK string) error {
	if langIn == "" || verSDK == "" {
		return nil
	}

	lang := LangLong[langIn] // Convert "py" to "python".
	if lang == "" {
		lang = langIn
	}

	verSDKCur, exists := VersionSDKsByName[lang]
	if !exists {
		return fmt.Errorf("CheckVerSDK, unknown lang: %s", lang)
	}

	verSDK = "v" + verSDK
	verSDKCur = "v" + verSDKCur

	if semver.Compare(semver.Major(verSDK), semver.Major(verSDKCur)) != 0 {
		return fmt.Errorf("CheckVerSDK, lang: %s,"+
			" wanted verSDK: %s major version is incompatible with verSDKCur: %s",
			lang, verSDK, verSDKCur)
	}

	if semver.Compare(semver.MajorMinor(verSDK), semver.MajorMinor(verSDKCur)) > 0 {
		return fmt.Errorf("CheckVerSDK, lang: %s,"+
			" wanted verSDK: %s is incompatible with verSDKCur: %s",
			lang, verSDK, verSDKCur)
	}

	return nil
}

// ------------------------------------------------

// Returns true if the wanted server version is compatible.
func CheckVerServer(verServer string) error {
	if verServer == "" {
		return nil
	}

	verServer = "v" + verServer

	// Convert from "7.0.0-4602-enterprise" to "v7.0.0".
	verServerCur := "v" + strings.Split(*version, "-")[0]

	if semver.Compare(semver.Major(verServer), semver.Major(verServerCur)) != 0 {
		return fmt.Errorf("CheckVerServer,"+
			" wanted verServer: %s major version is incompatible with verServerCur: %s",
			verServer, verServerCur)
	}

	if semver.Compare(semver.MajorMinor(verServer), semver.MajorMinor(verServerCur)) > 0 {
		return fmt.Errorf("CheckVerServer,"+
			" wanted verServer: %s is incompatible with verServerCur: %s",
			verServer, verServerCur)
	}

	return nil
}

// Get Encrypted Text
func HttpHandleET(w http.ResponseWriter, r *http.Request) {
	StatsNumInc("http.ET")

	cleartext := r.FormValue("ct")
	etext := r.FormValue("et")
	result := ""
	if cleartext != "" {
		result = Encrypt(*encryptKey, cleartext)
	}
	if etext != "" {
		result = Decrypt(*encryptKey, etext)
	}

	StatsNumInc("http.ET.ok")

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte(result))
}
