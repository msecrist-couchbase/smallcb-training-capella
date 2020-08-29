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

	static = flag.String("static", "cmd/play-server/static",
		"path to the 'static' resources directory")

	listen = flag.String("listen", ":8080",
		"HTTP listen [address]:port")

	workersMaxDuration = flag.Duration("workersMaxDuration", 20*time.Second,
		"max duration that a client's request will wait for a ready worker")

	workers = flag.Int("workers", 1,
		"# of workers (container instances)")

	restarters = flag.Int("restarters", 1,
		"# of restarters of the container instances")

	workersCh chan int // Channel of workerId's / container num's.

	restarterCh chan int // Channel of workerId's / container num's.

	langs = [][]string{
		// Tuple of [ lang (file suffix),
		//            langName (for display),
		//            exec command prefix (optional) ].
		[]string{"java", "java", "/run-java.sh"},
		[]string{"py", "python3", ""},
	}

	langNames = map[string]string{} // Map from 'py' to 'python3'.
	langExecs = map[string]string{} // Map from 'py' to exec command prefix.

	// Ex: { "basic-py": { "lang": "py", "code": "..." }, ... }.
	examples map[string]map[string]interface{}

	// Ex: [ "basic-py", "basic-java", ... ].
	exampleNames []string
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

	var err error

	examples, err = readYamls(*static + "/examples")
	if err != nil {
		log.Fatal(err)
	}

	for name := range examples {
		exampleNames = append(exampleNames, name)
	}

	sort.Strings(exampleNames)

	// Fill the workersCh with workerId tokens.
	workersCh = make(chan int, *workers)
	for i := 0; i < *workers; i++ {
		workersCh <- i
	}

	// The restarterCh is created with a capacity equal to
	// the # of workers to reduce workers having to wait.
	restarterCh = make(chan int, *workers)
	for i := 0; i < *restarters; i++ {
		go restarter(i, restarterCh, workersCh)
	}

	mux := http.NewServeMux()

	initMux(mux)

	log.Printf("listening on... %v", *listen)

	log.Fatal(http.ListenAndServe(*listen, mux))
}

// ------------------------------------------------

func initMux(mux *http.ServeMux) {
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(*static))))

	mux.HandleFunc("/run", handleRun)

	mux.HandleFunc("/", handleHome)
}

// ------------------------------------------------

func handleHome(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	lang := r.FormValue("lang")
	code := r.FormValue("code")

	mainTemplateEmit(w, name, lang, code)
}

// ------------------------------------------------

func handleRun(w http.ResponseWriter, r *http.Request) {
	lang := r.FormValue("lang")
	code := r.FormValue("code")

	output, err := runLangCode(r.Context(), lang, code)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", runLangCode, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("runLangCode, err: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.Write([]byte(output))
}

// ------------------------------------------------

func runLangCode(ctx context.Context, lang, code string) (
	string, error) {
	if lang == "" || code == "" {
		return "", nil
	}

	if len(code) > *codeMaxLen {
		return "", fmt.Errorf("code too long, codeMaxLen: %d", *codeMaxLen)
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

	case <-time.After(*workersMaxDuration):
		return "", fmt.Errorf("timeout waiting for worker, duration: %v", *workersMaxDuration)

	case <-ctx.Done():
		// Client canceled/timed-out while we were waiting.
		return "", ctx.Err()
	}

	// A worker is ready & assigned, so prepare the code dir & file.
	dir := fmt.Sprintf("%s%d", *containerVolPrefix, workerId)

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
	containerName := fmt.Sprintf("%s%d", *containerNamePrefix, workerId)

	var cmd *exec.Cmd

	execCommand := langExecs[lang]
	if len(execCommand) > 0 {
		// Case when there's an execCommand prefix,
		// such as "/run-java.sh .../tmp/play/code.java".
		cmd = exec.Command("docker", "exec", containerName,
			execCommand, codePathInst)
	} else {
		cmd = exec.Command("docker", "exec", containerName,
			codePathInst)
	}

	log.Printf("running cmd: %v\n", cmd)

	stdOutErr, err := execCmd(ctx, cmd, *codeMaxDuration)

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

type mainTemplateData struct {
	NameTitles []NameTitle
	Name       string
	Title      string
	Lang       string // Ex: 'py'.
	Code       string
	Output     string
}

func mainTemplateEmit(w http.ResponseWriter,
	name, lang, code string) {
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

	var title string

	if name != "" {
		c := examples[name]
		if c != nil {
			title = c["title"].(string)
			lang = c["lang"].(string)
			code = c["code"].(string)
		} else {
			mainTemplateEmit(w, "basic-py", "", "")
			return
		}
	}

	data := &mainTemplateData{
		NameTitles: nameTitles,
		Name:       name,
		Title:      title,
		Lang:       lang,
		Code:       code,
	}

	t, err := template.ParseFiles(*static + "/main.html.template")
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", template.ParseFiles, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("template.ParseFiles, err: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = t.Execute(w, data)
	if err != nil {
		log.Printf("t.Execute, data: %+v, err: %v", data, err)
	}
}

// ------------------------------------------------

func restarter(restarterId int, needRestartCh, doneRestartCh chan int) {
	for workerId := range needRestartCh {
		start := time.Now()

		log.Printf("restarterId: %d, workerId: %d\n",
			restarterId, workerId)

		cmd := exec.Command("make",
			fmt.Sprintf("CONTAINER_NUM=%d", workerId),
			"restart")

		stdOutErr, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("restarterId: %d, workerId: %d,"+
				" cmd: %v, stdOutErr: %v, err: %v",
				restarterId, workerId, cmd, stdOutErr, err)

			// Async try to restart the workerId again.
			go func(workerId int) { needRestartCh <- workerId }(workerId)

			continue
		}

		log.Printf("restarterId: %d, workerId: %d, took: %s\n",
			restarterId, workerId, time.Since(start))

		doneRestartCh <- workerId
	}
}

// ------------------------------------------------

// Run a cmd, waiting for it to finish or timeout, returning its
// combined stdout and stderr result.
func execCmd(ctx context.Context, cmd *exec.Cmd, duration time.Duration) (
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

func readYamls(dir string) (
	map[string]map[string]interface{}, error) {
	rv := map[string]map[string]interface{}{}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadDir, dir: %s, err: %v", dir, err)
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".yaml") {
			b, err := ioutil.ReadFile(dir + "/" + f.Name())
			if err != nil {
				return nil, fmt.Errorf("ioutil.ReadFile, f: %+v, err: %v", f, err)
			}

			m := make(map[string]interface{})

			err = yaml.Unmarshal(b, &m)
			if err != nil {
				return nil, fmt.Errorf("yaml.Unmarshal, f: %+v, err: %v", f, err)
			}

			rv[f.Name()[:len(f.Name())-5]] = m
		}
	}

	return rv, nil
}
