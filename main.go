package main

import (
	"log"
	"net/http"
	"os"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<p>Hello rabbits</p>"))
}

func staticLoader(w http.ResponseWriter, r *http.Request) {
	staticFilePath := "." + r.RequestURI

	dat, err := os.ReadFile(staticFilePath) // 💀💀💀 there may be some issues
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Printf("Requested nonexistent/failed to load static file %s", staticFilePath)
	} else {
		w.Write(dat)
		log.Printf("Served static file at location %s", staticFilePath)
	}
}

/* TODO: make recursive file search so that users can only request files
that match what is inside static folder (staticLoader). As its just
my personal project, I can live without it, but its a huge risk otherwise */
/* func staticFilesIndexer() map[string]bool{
	files, err := os.ReadDir("./static/")
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		if file.IsDir() {}
		log.Println(file.Name())
	}
} */

func main() {
	port := ":8080"

	router := http.NewServeMux()
	router.HandleFunc("GET /", indexHandler)
	router.HandleFunc("GET /{word}", indexHandler)
	router.HandleFunc("GET /static/", staticLoader)

	server := http.Server{
		Addr:    port,
		Handler: router,
	}

	log.Printf("Starting server http://localhost%s", server.Addr)
	server.ListenAndServe()
}
