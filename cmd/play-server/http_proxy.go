package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func HttpProxy(listenProxy string) {
	proxyMux := http.NewServeMux()

	proxyMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var sessionId string

		user, pswd, ok := r.BasicAuth()
		if ok {
			sessionId = user + pswd

			log.Printf("INFO: HttpProxy, path: %s, sessionId: %s,"+
				" via BasicAuth", r.URL.Path, sessionId)
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

			sessionId = user + pswd

			log.Printf("INFO: HttpProxy, path: %s, sessionId: %s,"+
				" via uilogin", r.URL.Path, sessionId)
		} else {
			for _, cookie := range r.Cookies() {
				c := cookie.Name + "=" + cookie.Value

				sessionId = CookiesGet(c)
				if sessionId != "" {
					log.Printf("INFO: HttpProxy, path: %s, sessionId: %s,"+
						" via cookie", r.URL.Path, sessionId)

					break
				}
			}
		}

		// Default to targetPort of 10001 so that we can
		// serve the web login UI without any auth or session.
		targetPort := *containerPublishPortBase + 1

		var modifyResponse func(response *http.Response) error

		if sessionId != "" {
			session := sessions.SessionGet(sessionId)
			if session == nil {
				http.Error(w,
					http.StatusText(http.StatusNotFound)+
						fmt.Sprintf(", HttpProxy, session not found"),
					http.StatusNotFound)
				log.Printf("ERROR: HttpProxy, path: %s, sessionId: %s,"+
					" session not found", r.URL.Path, sessionId)
				return
			}

			if session.ContainerId < 0 {
				http.Error(w,
					http.StatusText(http.StatusNotFound)+
						fmt.Sprintf(", HttpProxy, session w/o container"),
					http.StatusNotFound)
				log.Printf("ERROR: HttpProxy, path: %s, sessionId: %s,"+
					" session w/o container", r.URL.Path, sessionId)
				return
			}

			// Example targetPort: 10000 + (100 * containerId) + 1 == 10001.
			targetPort = *containerPublishPortBase +
				(*containerPublishPortSpan * session.ContainerId) + 1

			log.Printf("INFO: HttpProxy, path: %s, sessionId: %s, session ok,"+
				" containerId: %d, targetPort: %d", r.URL.Path, sessionId,
				session.ContainerId, targetPort)

			modifyResponse = func(response *http.Response) error {
				for _, cookie := range response.Cookies() {
					c := cookie.Name + "=" + cookie.Value

					CookiesSet(c, sessionId)
				}

				return nil
			}
		}

		// We can reach this point with a session, or reach here
		// session-less in the case of the web login UI screen.

		director := func(req *http.Request) {
			origin, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d/", targetPort))

			req.URL.Scheme = origin.Scheme
			req.URL.Host = origin.Host
		}

		proxy := &httputil.ReverseProxy{
			Director:       director,
			ModifyResponse: modifyResponse,
		}

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

	return ioutil.NopCloser(&buf),
		ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
