package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	initMux(mux)

	log.Printf("listening on port: %v", port)

	log.Fatal(http.ListenAndServe(":"+port, mux))
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
	code := r.FormValue("code")

	tmpDir, err := ioutil.TempDir("", "sandbox")
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(" - ioutil.TempDir, err: %v", err),
			http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	emit(w, r, &mainData{
		Code:   code,
		Output: "output would go here, but still TBD",
	})
}

type mainData struct {
	Code   string
	Output string
}

var mainTemplate = template.Must(template.ParseFiles("cmd/play-server/main.html.template"))

func emit(w http.ResponseWriter, r *http.Request, data *mainData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := mainTemplate.Execute(w, data); err != nil {
		log.Printf("mainTemplate.Execute, data: %+v, err: %v",
			data, err)
	}
}
