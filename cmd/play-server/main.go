package main

import (
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
)

var (
	h = flag.Bool("h", false, "help/usage info")

	help = flag.Bool("help", false, "help/usage info")

	langDefault = flag.String("langDefault", "py",
		"default programming lang (e.g., code file suffix)")

	maxCodeLen = flag.Int("maxCodeLen", 16000,
		"max allowed length of request code in bytes")

	containerPrefix = flag.String("containerPrefix", "smallcb-",
		"prefix of container instance name")

	volPrefix = flag.String("volPrefix", "vol-",
		"prefix of container instance volume directory")

	static = flag.String("static", "cmd/play-server/static",
		"path to the 'static' resources directory")

	listen = flag.String("listen", ":8080",
		"HTTP listen [address]:port")

	workers = flag.Int("workers", 1,
		"# of workers (containers) supported")

	workersCh chan int

	langPairs = [][]string{
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
	for _, item := range langPairs {
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

	workersN := *workers
	if workersN < 1 {
		workersN = 1
	}

	workersCh = make(chan int, workersN)
	for i := 0; i < workersN; i++ {
		workersCh <- i
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
	mainTemplateEmit(w, "", "", "")
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

func runLangCode(context context.Context, lang, code string) (
	string, error) {
	if lang == "" || code == "" {
		return "", nil
	}

	if len(code) > *maxCodeLen {
		return "", fmt.Errorf("code too long")
	}

	var workerId int

	select {
	case workerId = <-workersCh:
		defer func() {
			if workerId >= 0 {
				workersCh <- workerId
			}
		}()
	case <-context.Done():
		return "", nil
	}

	dir := fmt.Sprintf("%s%d", *volPrefix, workerId)

	err := os.MkdirAll(dir+"/tmp/play", 0777)
	if err != nil {
		return "", err
	}

	codePathHost := dir + "/tmp/play/code." + lang
	codePathInst := "/opt/couchbase/var/tmp/play/code." + lang

	codeBytes := []byte(strings.ReplaceAll(code, "\r\n", "\n"))

	// Executable in case of a script like 'code.py'.
	err = ioutil.WriteFile(codePathHost, codeBytes, 0777)
	if err != nil {
		return "", err
	}

	containerName := fmt.Sprintf("%s%d", *containerPrefix, workerId)

	var cmd *exec.Cmd

	execCommand := langExecs[lang]
	if len(execCommand) > 0 {
		cmd = exec.Command("docker", "exec", containerName,
			execCommand, codePathInst)
	} else {
		cmd = exec.Command("docker", "exec", containerName,
			codePathInst)
	}

	fmt.Printf("running cmd: %v\n", cmd)

	stdOutErr, err := cmd.CombinedOutput()

	// TODO: asynchronously restart the worker / smallcb-${workerId}.

	return string(stdOutErr), err
}

var mainTemplate = template.Must(template.ParseFiles(
	*static + "/main.html.template"))

type mainTemplateData struct {
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
