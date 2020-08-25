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
)

var (
	h = flag.Bool("h", false, "help/usage info")

	help = flag.Bool("help", false, "help/usage info")

	langDefault = flag.String("langDefault", "py",
		"default programming lang (e.g., file suffix)")

	maxCodeLen = flag.Int("maxCodeLen", 16000,
		"max allowed length of request code in bytes")

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
		[]string{"py", "python"}, // Pair of lang (file suffix) and langName.
	}

	langNames = map[string]string{} // Map from 'py' to 'python'.
	langCodes = map[string]string{} // Map from 'py' to example python code.
)

func init() {
	for _, langPair := range langPairs {
		lang, langName := langPair[0], langPair[1]

		langCode, err :=
			ioutil.ReadFile(*static + "/lang-code." + langPair[0])
		if err != nil {
			log.Fatalf("ioutil.ReadFile, lang: %s, err: %v",
				lang, err)
		}

		langNames[lang] = langName
		langCodes[lang] = string(langCode)
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

	langCode := r.FormValue("langCode")
	if len(langCode) > *maxCodeLen {
		mainTemplateEmit(w, lang, langCode, "ERROR: code too long")
		return
	}

	output, err := runLangCode(r.Context(), lang, langCode)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(" - err: %v", err),
			http.StatusInternalServerError)
	}

	mainTemplateEmit(w, lang, langCode, output)
}

func runLangCode(context context.Context, lang, langCode string) (
	string, error) {
	if lang == "" || langCode == "" {
		return "", nil
	}

	tmpDir, err := ioutil.TempDir("", "sandbox")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	var workerId int

	select {
	case workerId = <-workersCh:
		defer func() { workersCh <- workerId }()
	case <-context.Done():
		return "", nil
	}

	dir := fmt.Sprintf("%s%d", *volPrefix, workerId)

	err = os.Mkdir(dir+"/tmp/play", 0777)
	if err != nil {
		return "", err
	}

	return "output would go here / TODO\n", nil
}

var mainTemplate = template.Must(template.ParseFiles(*static + "/main.html.template"))

type mainTemplateData struct {
	Lang     string // Ex: 'py'.
	LangName string // Ex: 'python'.
	LangCode string
	Output   string
}

func mainTemplateEmit(w http.ResponseWriter,
	lang, langCode, output string) {
	if lang == "" {
		lang = *langDefault
	}

	if langCode == "" {
		langCode, _ = langCodes[lang]
	}

	data := &mainTemplateData{
		Lang:     lang,
		LangName: langNames[lang],
		LangCode: langCode,
		Output:   output,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := mainTemplate.Execute(w, data); err != nil {
		log.Printf("mainTemplate.Execute, data: %+v, err: %v",
			data, err)
	}
}
