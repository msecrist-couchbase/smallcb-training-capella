package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	h = flag.Bool("h", false, "print help/usage and exit")

	help = flag.Bool("help", false, "print help/usage and exit")

	codeMaxLen = flag.Int("codeMaxLen", 16000,
		"max length of a client's request code in bytes")

	codeMaxDuration = flag.Duration("codeMaxDuration", 10*time.Second,
		"max duration that a client's request code is allowed to run")

	containerNamePrefix = flag.String("containerNamePrefix", "smallcb-",
		"prefix of the names of container instances")

	containerVolPrefix = flag.String("containerVolPrefix", "vol-",
		"prefix of the volume directories of container instances")

	containerPortBase = flag.Int("containerPortBase", 10000,
		"base or starting port # for container instances")

	containerPortSpan = flag.Int("containerPortSpan", 100,
		"number of port #'s allocated for each container instance")

	containerAddr = flag.String("containerAddr", "127.0.0.1",
		"addr for publishing container instance ports")

	listen = flag.String("listen", ":8080",
		"HTTP listen [address]:port")

	static = flag.String("static", "cmd/play-server/static",
		"path to the 'static' resources directory")

	workersMaxDuration = flag.Duration("workersMaxDuration", 20*time.Second,
		"max duration that a client's request will wait for a ready worker")

	workers = flag.Int("workers", 1,
		"# of workers (container instances)")

	restarters = flag.Int("restarters", 1,
		"# of restarters of the container instances")

	// -----------------------------------

	TitleDefault = "API / SDK Playground"

	// -----------------------------------

	workersCh chan int // Channel of workerId's / container num's.

	restarterCh chan int // Channel of workerId's / container num's.

	// -----------------------------------

	// The langs is a config table with entries of...
	//   [ lang (code file suffix),
	//     langName (for UI display),
	//     execPrefix (exec command prefix for executing code; "" is ok) ].
	langs = [][]string{
		[]string{"java", "java", "/run-java.sh"},
		[]string{"py", "python3", ""},
	}

	langNames = map[string]string{} // Map from 'py' to 'python3'.
	langExecs = map[string]string{} // Map from 'py' to execPrefix.

	// -----------------------------------

	// Port mapping of container port # to containerPortBase + delta.
	portMapping = [][]int{
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

func init() {
	for _, item := range langs {
		lang, langName, langExec := item[0], item[1], item[2]

		langNames[lang] = langName
		langExecs[lang] = langExec
	}
}

// ------------------------------------------------

func main() {
	flag.Parse()

	if *h || *help {
		flag.Usage()
		os.Exit(2)
	}

	// The workersCh and restarterCh are created with capacity
	// equal to the # of workers to reduce the chance of
	// workers and restarters from having to wait.

	workersCh = make(chan int, *workers)

	restarterCh = make(chan int, *workers)

	// Spawn the restarter goroutines.
	for i := 0; i < *restarters; i++ {
		go Restarter(i, restarterCh, workersCh,
			*containerAddr, *containerPortBase, *containerPortSpan,
			portMapping)
	}

	// Have the restarters restart the required # of workers.
	for workerId := 0; workerId < *workers; workerId++ {
		restarterCh <- workerId
	}

	mux := http.NewServeMux()

	HttpMuxInit(mux)

	log.Printf("INFO: listening on... %v", *listen)

	log.Fatal(http.ListenAndServe(*listen, mux))
}

// ------------------------------------------------

func HttpMuxInit(mux *http.ServeMux) {
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(*static))))

	mux.HandleFunc("/run", HttpHandleRun)

	mux.HandleFunc("/", HttpHandleMain)
}

// ------------------------------------------------

func HttpHandleMain(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")

	if strings.HasPrefix(r.URL.Path, "/example/") &&
		len(r.URL.Path) > len("/example/") {
		name = r.URL.Path[len("/example/"):]
	}

	lang := r.FormValue("lang")
	code := r.FormValue("code")

	MainTemplateEmit(w, *static, *static+"/examples", name, lang, code)
}

// ------------------------------------------------

func HttpHandleRun(w http.ResponseWriter, r *http.Request) {
	lang := r.FormValue("lang")
	code := r.FormValue("code")

	output, err := RunLangCode(r.Context(), lang, code,
		*codeMaxLen, *codeMaxDuration, *workersMaxDuration,
		*containerNamePrefix, *containerVolPrefix)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", RunLangCode, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: RunLangCode, err: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.Write([]byte(output))
}

// ------------------------------------------------

func RunLangCode(ctx context.Context, lang, code string,
	codeMaxLen int, codeMaxDuration, workersMaxDuration time.Duration,
	containerNamePrefix, containerVolPrefix string) (
	string, error) {
	if lang == "" || code == "" {
		return "", nil
	}

	if len(code) > codeMaxLen {
		return "", fmt.Errorf("code too long, codeMaxLen: %d", codeMaxLen)
	}

	// Atomically grab a workerId token, blocking & waiting until
	// one is available.
	var workerId int

	select {
	case workerId = <-workersCh:
		defer func() {
			// Put the token back for the next request
			// handler if we still have it.
			if workerId >= 0 {
				workersCh <- workerId
			}
		}()

	case <-time.After(workersMaxDuration):
		return "", fmt.Errorf("timeout waiting for worker, duration: %v", workersMaxDuration)

	case <-ctx.Done():
		// Client canceled/timed-out while we were waiting.
		return "", ctx.Err()
	}

	// A worker is ready & assigned, so prepare the code dir & file.
	dir := fmt.Sprintf("%s%d", containerVolPrefix, workerId)

	err := os.MkdirAll(dir+"/tmp/play", 0777)
	if err != nil {
		return "", err
	}

	// Ex: "vol-0/tmp/play/code.py".
	codePathHost := dir + "/tmp/play/code." + lang

	// Ex: "/opt/couchbase/var/tmp/play/code.py".
	codePathInst := "/opt/couchbase/var/tmp/play/code." + lang

	codeBytes := []byte(strings.ReplaceAll(code, "\r\n", "\n"))

	// Mode is 0777 executable, for scripts like 'code.py'.
	err = ioutil.WriteFile(codePathHost, codeBytes, 0777)
	if err != nil {
		return "", err
	}

	// Ex: "smallcb-0".
	containerName := fmt.Sprintf("%s%d", containerNamePrefix, workerId)

	var cmd *exec.Cmd

	execPrefix := langExecs[lang]
	if len(execPrefix) > 0 {
		// Case of an execPrefix like "/run-java.sh".
		cmd = exec.Command("docker", "exec",
			"-u", "couchbase:couchbase",
			containerName, execPrefix, codePathInst)
	} else {
		cmd = exec.Command("docker", "exec",
			"-u", "couchbase:couchbase",
			containerName, codePathInst)
	}

	log.Printf("INFO: running cmd: %v\n", cmd)

	stdOutErr, err := ExecCmd(ctx, cmd, codeMaxDuration)

	select {
	case restarterCh <- workerId:
		// The restarter now owns the workerId token.
		workerId = -1
	case <-ctx.Done():
		return "", nil
	}

	return string(stdOutErr), err
}

// ------------------------------------------------

type NameTitle struct {
	Name, Title string
}

type MainTemplateData struct {
	NameTitles []NameTitle
	Name       string
	Title      string
	Lang       string // Ex: 'py'.
	Code       string
	Output     string
}

func MainTemplateEmit(w http.ResponseWriter,
	staticDir, examplesDir, name, lang, code string) {
	examples, exampleNames, err := ReadExamples(examplesDir)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", ReadExamples, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: ReadExamples, err: %v", err)
		return
	}

	nameTitles := make([]NameTitle, 0, len(exampleNames))
	for _, name := range exampleNames {
		title, _ := examples[name]["title"].(string)
		if title == "" {
			title = name
		}

		nameTitles = append(nameTitles, NameTitle{
			Name:  name,
			Title: title,
		})
	}

	if name == "" && lang == "" && code == "" {
		name = "basic-py"
	}

	var title string

	example := examples[name]
	if example != nil {
		if title == "" {
			title = example["title"].(string)
		}

		if lang == "" {
			lang = example["lang"].(string)
		}

		if code == "" {
			code = example["code"].(string)
		}
	}

	if title == "" {
		title = TitleDefault
	}

	data := &MainTemplateData{
		NameTitles: nameTitles,
		Name:       name,
		Title:      title,
		Lang:       lang,
		Code:       code,
	}

	t, err := template.ParseFiles(staticDir + "/main.html.template")
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", template.ParseFiles, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: template.ParseFiles, err: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = t.Execute(w, data)
	if err != nil {
		log.Printf("ERROR: t.Execute, data: %+v, err: %v", data, err)
	}
}

// ------------------------------------------------

func Restarter(restarterId int, needRestartCh, doneRestartCh chan int,
	containerAddr string, containerPortBase, containerPortSpan int,
	portMapping [][]int) {
	for workerId := range needRestartCh {
		start := time.Now()

		cmd := exec.Command("make",
			fmt.Sprintf("CONTAINER_NUM=%d", workerId))

		portBase := containerPortBase + (containerPortSpan * workerId)

		ports := make([]string, 0, len(portMapping))
		for _, port := range portMapping {
			ports = append(ports,
				fmt.Sprintf("-p %s:%d:%d/tcp",
					containerAddr, portBase+port[1], port[0]))
		}

		cmd.Args = append(cmd.Args,
			"CONTAINER_PORTS="+strings.Join(ports, " "))

		cmd.Args = append(cmd.Args, "restart")

		log.Printf("INFO: restarterId: %d, workerId: %d\n",
			restarterId, workerId)

		stdOutErr, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("ERROR: restarterId: %d, workerId: %d,"+
				" cmd: %v, stdOutErr: %s, err: %v",
				restarterId, workerId, cmd, stdOutErr, err)

			// Async try to restart the workerId again.
			go func(workerId int) { needRestartCh <- workerId }(workerId)

			continue
		}

		log.Printf("INFO: restarterId: %d, workerId: %d, took: %s\n",
			restarterId, workerId, time.Since(start))

		doneRestartCh <- workerId
	}
}

// ------------------------------------------------

// Run a cmd, waiting for it to finish or timeout, returning its
// combined stdout and stderr result.
func ExecCmd(ctx context.Context, cmd *exec.Cmd, duration time.Duration) (
	[]byte, error) {
	var b bytes.Buffer

	cmd.Stdout = &b
	cmd.Stderr = &b

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cmd.Start, err: %v", err)
	}

	doneCh := make(chan error, 1)
	go func() {
		doneCh <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		return nil, fmt.Errorf("ctx.Done, err: %v", ctx.Err())

	case <-time.After(duration):
		cmd.Process.Kill()
		return nil, fmt.Errorf("timeout, duration: %v", duration)

	case err := <-doneCh:
		if err != nil {
			return nil, fmt.Errorf("doneCh, err: %v", err)
		}
	}

	return b.Bytes(), nil
}

// ------------------------------------------------

// ReadExamples will return...
//   examples:
//     { "basic-py": { "title": "...", "lang": "py", "code": "..." }, ... }.
//   exampleNames (sorted ASC):
//     [ "basic-py", ... ].
func ReadExamples(dir string) (
	examples map[string]map[string]interface{},
	exampleNames []string,
	err error) {
	examples, err = ReadYamls(dir, ".yaml")
	if err != nil {
		return nil, nil, err
	}

	for name, example := range examples {
		// Only yaml's with a title are considered examples.
		if _, hasTitle := example["title"]; hasTitle {
			exampleNames = append(exampleNames, name)
		}
	}

	sort.Strings(exampleNames)

	return examples, exampleNames, nil
}

// ------------------------------------------------

func ReadYamls(dir, suffix string) (
	map[string]map[string]interface{}, error) {
	rv := map[string]map[string]interface{}{}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadDir, dir: %s, err: %v", dir, err)
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), suffix) {
			m, err := ReadYaml(dir + "/" + f.Name())
			if err != nil {
				return nil, fmt.Errorf("ReadYaml, f: %+v, err: %v", f, err)
			}

			rv[f.Name()[:len(f.Name())-len(suffix)]] = m
		}
	}

	return rv, nil
}

func ReadYaml(path string) (map[string]interface{}, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadFile, path: %s, err: %v", path, err)
	}

	m := make(map[string]interface{})

	err = yaml.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("yaml.Unmarshal, path: %s, err: %v", path, err)
	}

	return m, nil
}
