package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

func HttpProxy(listenProxy string, portMap map[int]int,
	containerPublishPortBase int,
	containerPublishPortSpan int) {
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
		targetPort := containerPublishPortBase + 1

		var modifyResponse func(response *http.Response) error

		var flushInterval time.Duration

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

			// Example portStart: 10000 + (100 * containerId) == 10000.
			portStart := containerPublishPortBase +
				(containerPublishPortSpan * session.ContainerId)

			// Example targetPort: portStart + 1 == 10001.
			targetPort = portStart + portMap[8091]

			streamingJson, remap := IsStreamingJsonURLPath(r.URL.Path)

			modifyResponse = func(resp *http.Response) (err error) {
				for _, cookie := range resp.Cookies() {
					c := cookie.Name + "=" + cookie.Value

					CookiesSet(c, sessionId)
				}

				if streamingJson {
					resp.Body = &jsonStreamer{
						remap:     remap,
						portMap:   portMap,
						portStart: portStart,
						src:       resp.Body,
						srcReader: bufio.NewReader(resp.Body),
					}
				}

				return nil
			}

			if streamingJson {
				flushInterval = 2 * time.Second
			}

			log.Printf("INFO: HttpProxy, path: %s, sessionId: %s,"+
				" containerId: %d, streamingJson: %t, remap: %t",
				r.URL.Path, sessionId, session.ContainerId,
				streamingJson, remap)
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
			FlushInterval:  flushInterval,
		}

		proxy.ServeHTTP(w, r)
	})

	log.Printf("INFO: HttpProxy, listenProxy: %s", listenProxy)

	log.Fatal(http.ListenAndServe(listenProxy, proxyMux))
}

// ------------------------------------------------

func IsStreamingJsonURLPath(path string) (bool, bool) {
	if strings.HasPrefix(path, "/poolsStreaming/") {
		return true, false
	}

	if strings.HasPrefix(path, "/pools/") {
		// Ex: "/pools/default/bs/beer-sample".
		parts := strings.Split(path, "/")
		if len(parts) > 4 && parts[3] == "bs" {
			return true, true
		}
	}

	// TODO: More streaming JSON URL paths (used by SDK's)?

	return false, false
}

// ------------------------------------------------

// Rewrites streaming JSON, which is JSON delimited by 4 newlines,
// optionally remapping port numbers.
type jsonStreamer struct {
	remap     bool
	portMap   map[int]int
	portStart int
	src       io.Closer
	srcReader *bufio.Reader
	out       bytes.Buffer
}

func (s *jsonStreamer) Close() error {
	return s.src.Close()
}

func (s *jsonStreamer) Read(p []byte) (n int, err error) {
	if s.out.Len() <= 0 {
		s.out.Reset()

		b, err := s.srcReader.ReadBytes('\n')
		if err != nil {
			s.src.Close()
			return 0, err
		}

		for i := 0; i < 3; i++ { // Read 3 newlines.
			nl, err := s.srcReader.ReadByte()
			if err != nil || nl != '\n' {
				s.src.Close()
				return 0, io.EOF
			}
		}

		var m map[string]interface{}

		err = json.Unmarshal(b, &m)
		if err != nil {
			s.src.Close()
			return 0, err
		}

		if s.remap {
			s.RemapJson(m)
		}

		b, err = json.Marshal(m)
		if err != nil {
			s.src.Close()
			return 0, err
		}

		b = append(b, '\n', '\n', '\n', '\n')

		s.out.Write(b)
	}

	return s.out.Read(p)
}

// ------------------------------------------------

func (s *jsonStreamer) RemapJson(m map[string]interface{}) {
	if v, exists := m["nodesExt"]; exists && v != nil {
		if va, ok := v.([]interface{}); ok && va != nil {
			for _, vav := range va {
				s.RemapJsonNodeExt(vav)
			}
		}
	}
}

func (s *jsonStreamer) RemapJsonNodeExt(vav interface{}) {
	if nodeExt, ok := vav.(map[string]interface{}); ok && nodeExt != nil {
		if v2, exists := nodeExt["services"]; exists && v2 != nil {
			s.RemapJsonServices(v2)
		}
	}
}

func (s *jsonStreamer) RemapJsonServices(v2 interface{}) {
	if services, ok := v2.(map[string]interface{}); ok && services != nil {
		for service, v3 := range services {
			if port, ok := v3.(float64); ok {
				if portDelta, exists := s.portMap[int(port)]; exists {
					services[service] = s.portStart + portDelta
				}
			}
		}
	}
}

// ------------------------------------------------

// DupBody is based on httputil.DrainBody, and reads all of b to
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
