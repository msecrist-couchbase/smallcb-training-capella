package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func HttpProxy(listenProxy, // Ex: ":8091", ":8093".
	staticDir string,
	proxyFlushInterval time.Duration,
	host string, // Ex: "couchbase.live", "127.0.0.1".
	portApp int, // Ex: 8080.
	portMap map[int]int,
	containerPublishPortBase int,
	containerPublishPortSpan int) {
	portProxy, _ := strconv.Atoi(strings.Split(listenProxy, ":")[1]) // Ex: 8091.

	proxyMux := http.NewServeMux()

	proxyMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var sessionId, sessionIdFrom string

		user, pswd, ok := r.BasicAuth()
		if ok {
			if user == "username" && pswd == "password" {
				http.Error(w,
					http.StatusText(http.StatusUnauthorized)+
						fmt.Sprintf(", HttpProxy, username/password rejected,"+
							" try using a test-drive session"),
					http.StatusUnauthorized)

				log.Printf("ERROR: HttpProxy, path: %s,"+
					" err: username/password rejected", r.URL.Path)

				return
			}

			sessionId = user + pswd
			if sessionId != "" {
				sessionIdFrom = "BasicAuth"
			}
		}

		if sessionId == "" &&
			r.URL.Path == "/uilogin" && r.Method == "POST" {
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
			if sessionId != "" {
				sessionIdFrom = "cookie"
			}
		}

		if sessionId == "" {
			for _, cookie := range r.Cookies() {
				c := cookie.Name + "=" + cookie.Value

				sessionId = CookiesGet(c)
				if sessionId != "" {
					sessionIdFrom = "cookie"

					break
				}
			}
		}

		// Default to targetPort of 10001 so that we can
		// serve the web login UI without any auth or session.
		targetPort := containerPublishPortBase + portMap[portProxy]

		var modifyResponse func(response *http.Response) error

		var flushInterval time.Duration

		if sessionId != "" {
			log.Printf("INFO: HttpProxy, port: %d,"+
				" path: %s, sessionId: %s, via %s",
				portProxy, r.URL.Path, sessionId, sessionIdFrom)

			session := sessions.SessionGet(sessionId)
			if session == nil {
				log.Printf("ERROR: HttpProxy, path: %s, unknown sessionId: %s",
					r.URL.Path, sessionId)

				sessionId = ""
				sessionIdFrom = ""
			} else if session.ContainerId < 0 {
				http.Error(w,
					http.StatusText(http.StatusNotFound)+
						fmt.Sprintf(", HttpProxy, session w/o container"),
					http.StatusNotFound)

				log.Printf("ERROR: HttpProxy, path: %s, sessionId: %s,"+
					" no container", r.URL.Path, sessionId)

				return
			} else {
				// Example portStart: 10000 + (100 * containerId) == 10000.
				portStart := containerPublishPortBase +
					(containerPublishPortSpan * session.ContainerId)

				// Example targetPort: portStart + 1 == 10001.
				targetPort = portStart + portMap[portProxy]

				remapResponse, streamResponse :=
					ResponseKindForURLPath(r.URL.Path)

				modifyResponse = func(resp *http.Response) (err error) {
					for _, cookie := range resp.Cookies() {
						c := cookie.Name + "=" + cookie.Value

						CookiesSet(c, sessionId)
					}

					if streamResponse {
						js := &JsonStreamer{
							src:       resp.Body,
							srcReader: bufio.NewReader(resp.Body),
						}

						if remapResponse {
							js.remapper = &JsonRemapper{
								host:      host,
								portMap:   portMap,
								portStart: portStart,
							}
						}

						resp.Body = js
					} else if remapResponse {
						err = RemapResponse(resp, &JsonRemapper{
							host:      host,
							portMap:   portMap,
							portStart: portStart,
						})
					} else if strings.HasPrefix(r.URL.Path, "/ui/index.html") {
						err = InjectResponseUI(staticDir,
							host, portApp, session, resp)
					}

					return err
				}

				if streamResponse {
					flushInterval = proxyFlushInterval
				}

				log.Printf("INFO: HttpProxy, port: %d, path: %s,"+
					" sessionId: %s, containerId: %d, remap: %t, stream: %t",
					portProxy, r.URL.Path, sessionId, session.ContainerId,
					remapResponse, streamResponse)
			}
		}

		if modifyResponse == nil &&
			strings.HasPrefix(r.URL.Path, "/ui/index.html") {
			modifyResponse = func(resp *http.Response) (err error) {
				return InjectResponseUI(staticDir,
					host, portApp, nil, resp)
			}
		}

		// We can reach this point with a session, or reach here
		// session-less in the case of the web login UI screen.

		director := func(req *http.Request) {
			origin, _ := url.Parse(
				fmt.Sprintf("http://127.0.0.1:%d/", targetPort))

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

func ResponseKindForURLPath(path string) (needsRemap, needsStreaming bool) {
	for len(path) > 0 && path[len(path)-1] == '/' {
		path = path[0 : len(path)-1] // Strip trailing '/'.
	}

	if strings.HasPrefix(path, "/poolsStreaming/") {
		return true, true
	}

	if strings.HasPrefix(path, "/pools/") {
		parts := strings.Split(path, "/")
		if len(parts) == 3 {
			return true, false // Ex: "/pools/default".
		}

		// Ex: "/pools/default/buckets".
		if len(parts) == 4 {
			if parts[3] == "buckets" {
				return true, false
			}
		}

		// Ex: "/pools/default/buckets|bucketsStreaming|bs/beer-sample".
		if len(parts) == 5 {
			if parts[3] == "buckets" {
				return true, false
			}

			if parts[3] == "bs" ||
				parts[3] == "bucketsStreaming" {
				return true, true
			}
		}
	}

	return false, false
}

// ------------------------------------------------

// Implements io.ReadCloser for streaming response JSON, which
// is JSON that's delimited by 4 newlines, with optional
// remapping of port numbers.
type JsonStreamer struct {
	remapper  *JsonRemapper
	src       io.ReadCloser
	srcReader *bufio.Reader
	out       bytes.Buffer
}

func (s *JsonStreamer) Close() error {
	return s.src.Close()
}

func (s *JsonStreamer) Read(p []byte) (n int, err error) {
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

		var m interface{}

		err = json.Unmarshal(b, &m)
		if err != nil {
			s.src.Close()
			return 0, err
		}

		if s.remapper != nil {
			s.remapper.RemapJson(m)
		}

		b, err = json.Marshal(m)
		if err != nil {
			s.src.Close()
			return 0, err
		}

		s.out.Write(b)
		s.out.Write([]byte("\n\n\n\n"))

		fmt.Printf("b: %s\n", s.out.Bytes())
	}

	return s.out.Read(p)
}

// ------------------------------------------------

func RemapResponse(resp *http.Response, remapper *JsonRemapper) (err error) {
	if resp.Body == nil || resp.Body == http.NoBody {
		// No copying needed. Preserve the magic sentinel of NoBody.
		return nil
	}

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(resp.Body); err != nil {
		return err
	}

	var m interface{}

	err = json.Unmarshal(buf.Bytes(), &m)
	if err != nil {
		return err
	}

	remapper.RemapJson(m)

	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	resp.Body = &ReaderCloser{
		reader: bytes.NewReader(b),
		closer: resp.Body,
	}

	resp.ContentLength = int64(len(b))

	resp.Header["Content-Length"] = []string{fmt.Sprintf("%d", len(b))}

	return nil
}

// ------------------------------------------------

func InjectResponseUI(staticDir string, host string, portApp int,
	session *Session, resp *http.Response) error {
	resp.Header.Del("X-Frame-Options")

	t := template.Must(template.ParseFiles(staticDir + "/inject.html.tmpl"))

	var tout bytes.Buffer

	err := t.Execute(&tout, SessionTemplateData(host, portApp, session,
		*listenPortBase, *listenPortSpan, PortMapping))
	if err != nil {
		return fmt.Errorf("t.Execute, err: %v", err)
	}

	tout.Write([]byte("</head>")) // Append head close tag.

	if resp.Body == nil || resp.Body == http.NoBody {
		// No copying needed. Preserve the magic sentinel of NoBody.
		return nil
	}

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(resp.Body); err != nil {
		return err
	}

	b := buf.Bytes()

	b = bytes.Replace(b, []byte("</head>"), tout.Bytes(), 1)

	resp.Body = &ReaderCloser{
		reader: bytes.NewReader(b),
		closer: resp.Body,
	}

	resp.ContentLength = int64(len(b))

	resp.Header["Content-Length"] = []string{fmt.Sprintf("%d", len(b))}

	return nil
}

// ------------------------------------------------

type JsonRemapper struct {
	host      string
	portMap   map[int]int
	portStart int
}

func (s *JsonRemapper) RemapJson(v interface{}) {
	if m, ok := v.(map[string]interface{}); ok {
		s.RemapJsonMap(m)
	}

	if a, ok := v.([]interface{}); ok {
		for _, vv := range a {
			s.RemapJson(vv)
		}
	}
}

func (s *JsonRemapper) RemapJsonMap(m map[string]interface{}) {
	if v, exists := m["nodes"]; exists && v != nil {
		if va, ok := v.([]interface{}); ok && va != nil {
			for _, vav := range va {
				s.RemapJsonNode(vav)
			}
		}
	}

	if v, exists := m["nodesExt"]; exists && v != nil {
		if va, ok := v.([]interface{}); ok && va != nil {
			for _, vav := range va {
				s.RemapJsonNodeExt(vav)
			}
		}
	}

	if v, exists := m["vBucketServerMap"]; exists && v != nil {
		if vm, ok := v.(map[string]interface{}); ok && vm != nil {
			if v1, exists := vm["serverList"]; exists && v1 != nil {
				s.RemapJsonServerList(v1)
			}
		}
	}
}

func (s *JsonRemapper) RemapJsonNode(vav interface{}) {
	if node, ok := vav.(map[string]interface{}); ok && node != nil {
		if v2, exists := node["hostname"]; exists && v2 != nil {
			if str, ok := v2.(string); ok && str != "$HOST:8091" {
				node["hostname"] = s.host + ":8091"
			}
		}
	}
}

func (s *JsonRemapper) RemapJsonNodeExt(vav interface{}) {
	if nodeExt, ok := vav.(map[string]interface{}); ok && nodeExt != nil {
		if v2, exists := nodeExt["services"]; exists && v2 != nil {
			s.RemapJsonServices(v2)
		}
	}
}

func (s *JsonRemapper) RemapJsonServices(v2 interface{}) {
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

// Remap ["$HOST:11210"] to ["$HOST:10030"].
func (s *JsonRemapper) RemapJsonServerList(v interface{}) {
	if serverList, ok := v.([]interface{}); ok && serverList != nil {
		for i, x := range serverList {
			if str, ok := x.(string); ok {
				parts := strings.Split(str, ":")
				if len(parts) == 2 {
					port, err := strconv.Atoi(parts[1])
					if err != nil {
						continue
					}

					portDelta, exists := s.portMap[port]
					if !exists {
						continue
					}

					host := s.host
					if parts[0] == "$HOST" {
						host = "$HOST"
					}

					serverList[i] = fmt.Sprintf("%s:%d",
						host, s.portStart+portDelta)
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

// ------------------------------------------------

type ReaderCloser struct {
	reader io.Reader
	closer io.Closer
}

func (s *ReaderCloser) Close() error {
	return s.closer.Close()
}

func (s *ReaderCloser) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
}
