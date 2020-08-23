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

	mux.HandleFunc("/", handleHome)
}

var homeTemplate = template.Must(template.ParseFiles("cmd/play-server/home.html"))

func handleHome(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := homeTemplate.Execute(w, data); err != nil {
		log.Printf("homeTemplate.Execute, data: %+v, err:%v",
			data, err)
	}
}
