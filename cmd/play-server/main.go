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
	listen = flag.String("listen", ":8080",
		"HTTP listen [address]:port")

	concurrency = flag.Int("concurrency", 1,
		"max # of concurrent run requests supported")

	concurrencyCh chan struct{}
)

func main() {
	concurrencyCh = make(chan struct{}, *concurrency)

	mux := http.NewServeMux()

	initMux(mux)

	log.Printf("listening on... %v", *listen)

	log.Fatal(http.ListenAndServe(*listen, mux))
}

func initMux(mux *http.ServeMux) {
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("cmd/play-server/static"))))

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
	code := r.FormValue("code")

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
	case concurrencyCh <- struct{}{}:
	case <-r.Context().Done():
		return nil
	}
	defer func() { <-concurrencyCh }()

	return &mainData{
		Code:   code,
		Output: "output would go here, but still TBD",
	}
}

type mainData struct {
	Code   string
	Output string
}

var mainTemplate = template.Must(template.ParseFiles("cmd/play-server/main.html.template"))

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
