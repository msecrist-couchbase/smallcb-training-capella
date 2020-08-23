package main

import (
	"html/template"
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

	log.Fatalf("err: %v", http.ListenAndServe(":"+port, mux))
}

func initMux(mux *http.ServeMux) {
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("cmd/play-server/static"))))

	mux.HandleFunc("/run", handleRun)

	mux.HandleFunc("/", handleHome)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	emit(w, r, &form{})
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")

	emit(w, r, &form{
		Code:   code,
		Output: "an exercise left to the reader",
	})
}

type form struct {
	Code   string
	Output string
}

var formTemplate = template.Must(template.ParseFiles("cmd/play-server/form.html.template"))

func emit(w http.ResponseWriter, r *http.Request, data *form) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := formTemplate.Execute(w, data); err != nil {
		log.Printf("formTemplate.Execute, data: %+v, err: %v",
			data, err)
	}
}
