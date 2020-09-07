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
	"strconv"
	"strings"
	"time"
)

func HttpProxy(listenProxy string,
	containerPublishHost string,
	portMap map[int]int,
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

			remapResponse, streamResponse :=
				ResponseKindForURLPath(r.URL.Path)

			modifyResponse = func(resp *http.Response) (err error) {
				for _, cookie := range resp.Cookies() {
					c := cookie.Name + "=" + cookie.Value

					CookiesSet(c, sessionId)
				}

				if streamResponse {
					log.Printf("MR SR, header: %+v", resp.Header)

					js := &JsonStreamer{
						src:       resp.Body,
						srcReader: bufio.NewReader(resp.Body),
					}

					if remapResponse {
						js.remapper = &JsonRemapper{
							host:      containerPublishHost,
							portMap:   portMap,
							portStart: portStart,
						}
					}

					resp.Body = js
				} else if remapResponse {
					err = RemapResponse(resp, &JsonRemapper{
						host:      containerPublishHost,
						portMap:   portMap,
						portStart: portStart,
					})
				}

				return err
			}

			if streamResponse {
				flushInterval = 200 * time.Millisecond
			}

			log.Printf("INFO: HttpProxy, path: %s, sessionId: %s,"+
				" containerId: %d, remap: %t, stream: %t",
				r.URL.Path, sessionId, session.ContainerId,
				remapResponse, streamResponse)
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

func ResponseKindForURLPath(path string) (needsRemap, needsStreaming bool) {
	if strings.HasPrefix(path, "/poolsStreaming/") {
		return true, true
	}

	if strings.HasPrefix(path, "/pools/") {
		parts := strings.Split(path, "/")
		if len(parts) == 3 {
			return true, false // Ex: "/pools/default".
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

		var m map[string]interface{}

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
	log.Printf("RemapResponse, header: %+v", resp.Header)

	if resp.Body == nil || resp.Body == http.NoBody {
		// No copying needed. Preserve the magic sentinel of NoBody.
		return nil
	}

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(resp.Body); err != nil {
		return err
	}

	var m map[string]interface{}

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

	log.Printf("remap-body: %s (%d)", b, len(b))

	return nil
}

// ------------------------------------------------

type JsonRemapper struct {
	host      string
	portMap   map[int]int
	portStart int
}

func (s *JsonRemapper) RemapJson(m map[string]interface{}) {
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

					serverList[i] = fmt.Sprintf("%s:%d",
						parts[0], s.portStart+portDelta)
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
	err := s.closer.Close()
	log.Printf("rc.Close, err: %v", err)
	return err
}

func (s *ReaderCloser) Read(p []byte) (n int, err error) {
	n, err = s.reader.Read(p)
	log.Printf("rc.Read %d => n: %d, err: %v", len(p), n, err)
	return n, err
}
