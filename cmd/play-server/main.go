package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var (
	static = flag.String("static", "cmd/play-server/static",
		"path to the 'static' resources directory")

	h = flag.Bool("h", false, "help/usage info")

	help = flag.Bool("help", false, "help/usage info")

	listen = flag.String("listen", ":8080",
		"HTTP listen [address]:port")

	concurrency = flag.Int("concurrency", 1,
		"max # of concurrent run requests supported")

	concurrencyCh chan int

	langs = []string{
		"py",
	}

	langCodes = map[string]string{}
)

func init() {
	for _, lang := range langs {
		code, err := ioutil.ReadFile(*static + "/lang-code." + lang)
		if err != nil {
			log.Fatalf("ioutil.ReadFile, lang: %s, err: %v",
				lang, err)
		}

		langCodes[lang] = string(code)
	}
}

func main() {
	flag.Parse()

	if *h || *help {
		flag.Usage()
		os.Exit(2)
	}

	concurrencyN := *concurrency
	if concurrencyN < 1 {
		concurrencyN = 1
	}

	concurrencyCh = make(chan int, concurrencyN)
	for i := 0; i < concurrencyN; i++ {
		concurrencyCh <- i
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
	emit(w, r, &mainData{})
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	emit(w, r, handleRunCode(w, r))
}

func handleRunCode(w http.ResponseWriter, r *http.Request) *mainData {
	lang := r.FormValue("lang")
	code := r.FormValue("code")

	if lang != "" && code != "" {
		tmpDir, err := ioutil.TempDir("", "sandbox")
		if err != nil {
			http.Error(w,
				http.StatusText(http.StatusInternalServerError)+
					fmt.Sprintf(" - ioutil.TempDir, err: %v", err),
				http.StatusInternalServerError)
			return nil
		}
		defer os.RemoveAll(tmpDir)

		// Bound the # of concurrent requests.
		select {
		case token := <-concurrencyCh:
			defer func() { concurrencyCh <- token }()
		case <-r.Context().Done():
			return nil
		}
	}

	if lang == "" {
		lang = "py"
	}

	if code == "" {
		code = langCodes[lang]
	}

	return &mainData{
		Lang:   lang,
		Code:   code,
		Output: "output would go here, but still TBD",
	}
}

type mainData struct {
	Lang   string
	Code   string
	Output string
}

var mainTemplate = template.Must(template.ParseFiles(*static + "/main.html.template"))

func emit(w http.ResponseWriter, r *http.Request, data *mainData) {
	if data == nil {
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := mainTemplate.Execute(w, data); err != nil {
		log.Printf("mainTemplate.Execute, data: %+v, err: %v",
			data, err)
	}
}
