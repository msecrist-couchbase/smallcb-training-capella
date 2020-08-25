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
	"strings"
	"time"
)

var (
	h = flag.Bool("h", false, "print help/usage and exit")

	help = flag.Bool("help", false, "print help/usage and exit")

	langDefault = flag.String("langDefault", "py",
		"default programming lang (e.g., code file suffix)")

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

	workersCh chan int

	restarterCh chan int

	langs = [][]string{
		// Tuple of [ lang (file suffix),
		//            langName,
		//            exec command prefix ].
		[]string{"java", "java", "/run-java.sh"},
		[]string{"py", "python3", ""},
	}

	langNames = map[string]string{} // Map from 'py' to 'python3'.
	langCodes = map[string]string{} // Map from 'py' to example python3 code.
	langExecs = map[string]string{} // Map from 'py' to exec command prefix.
)

func init() {
	for _, item := range langs {
		lang, langName, langExec := item[0], item[1], item[2]

		langCode, err :=
			ioutil.ReadFile(*static + "/lang-code." + lang)
		if err != nil {
			log.Fatalf("ioutil.ReadFile, lang: %s, err: %v",
				lang, err)
		}

		langNames[lang] = langName
		langCodes[lang] = string(langCode)
		langExecs[lang] = langExec
	}
}

func main() {
	flag.Parse()

	if *h || *help {
		flag.Usage()
		os.Exit(2)
	}

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

func initMux(mux *http.ServeMux) {
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(*static))))

	mux.HandleFunc("/run", handleRun)

	mux.HandleFunc("/", handleHome)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	lang := r.FormValue("lang")

	mainTemplateEmit(w, lang, "", "")
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	lang := r.FormValue("lang")
	code := r.FormValue("code")

	output, err := runLangCode(r.Context(), lang, code)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(" - err: %v", err),
			http.StatusInternalServerError)
		return
	}

	mainTemplateEmit(w, lang, code, output)
}

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

	dir := fmt.Sprintf("%s%d", *containerVolPrefix, workerId)

	err := os.MkdirAll(dir+"/tmp/play", 0777)
	if err != nil {
		return "", err
	}

	// Ex: "vol-0/tmp/play/code.py" path.
	codePathHost := dir + "/tmp/play/code." + lang

	// Ex: "/opt/couchbase/var/tmp/play/code.py" path.
	codePathInst := "/opt/couchbase/var/tmp/play/code." + lang

	codeBytes := []byte(strings.ReplaceAll(code, "\r\n", "\n"))

	// Mode is 0777 executable in case it's a script like 'code.py'.
	err = ioutil.WriteFile(codePathHost, codeBytes, 0777)
	if err != nil {
		return "", err
	}

	containerName := fmt.Sprintf("%s%d", *containerNamePrefix, workerId)

	var cmd *exec.Cmd

	execCommand := langExecs[lang]
	if len(execCommand) > 0 {
		// Case when there's an execCommand prefix.
		cmd = exec.Command("docker", "exec", containerName,
			execCommand, codePathInst)
	} else {
		cmd = exec.Command("docker", "exec", containerName,
			codePathInst)
	}

	fmt.Printf("running cmd: %v\n", cmd)

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

var mainTemplate = template.Must(template.ParseFiles(
	*static + "/main.html.template"))

type mainTemplateData struct {
	Langs    [][]string
	Lang     string // Ex: 'py'.
	LangName string // Ex: 'python'.
	Code     string
	Output   string
}

func mainTemplateEmit(w http.ResponseWriter,
	lang, code, output string) {
	if lang == "" {
		lang = *langDefault
	}

	if code == "" {
		code, _ = langCodes[lang]
	}

	data := &mainTemplateData{
		Langs:    langs,
		Lang:     lang,
		LangName: langNames[lang],
		Code:     code,
		Output:   output,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := mainTemplate.Execute(w, data); err != nil {
		log.Printf("mainTemplate.Execute, data: %+v, err: %v",
			data, err)
	}
}

// ------------------------------------------------

func restarter(restarterId int, needRestartCh, doneRestartCh chan int) {
	for workerId := range needRestartCh {
		start := time.Now()

		fmt.Printf("restarterId: %d, workerId: %d\n",
			restarterId, workerId)

		cmd := exec.Command("make",
			fmt.Sprintf("CONTAINER_NUM=%d", workerId),
			"restart")

		stdOutErr, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("restarterId: %d, cmd: %v, stdOutErr: %v, err: %v",
				restarterId, cmd, stdOutErr, err)
		}

		fmt.Printf("restarterId: %d, workerId: %d, took: %s\n",
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
		return nil, err
	}

	doneCh := make(chan error, 1)
	go func() {
		doneCh <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		return nil, ctx.Err()

	case <-time.After(duration):
		cmd.Process.Kill()
		return nil, fmt.Errorf("timeout, duration: %v", duration)

	case err := <-doneCh:
		if err != nil {
			return nil, err
		}
	}

	return b.Bytes(), nil
}
