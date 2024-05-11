package main

import (
	"bytes"
	"database/sql"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	Name string
}

type Word struct {
	Woord      string
	Woordsoort string
	Uitspraak  string
	Vertaling  string
}

func renderWordsTable(query string) []byte {
	t := template.Must(template.ParseFiles("./templates/table.html"))
	mockData := []Word{
		{Woord: "frikandel", Woordsoort: "nw", Uitspraak: "[фрикандель]", Vertaling: ""},
		{Woord: "stroopwafel", Woordsoort: "nw", Uitspraak: "[строопвафел]", Vertaling: ""},
		{Woord: "kaneelbroodje", Woordsoort: "nw", Uitspraak: "[канельброодье]", Vertaling: "булочка с корицей"},
	}
	var wordsTable bytes.Buffer
	t.Execute(&wordsTable, mockData)
	return wordsTable.Bytes()
}

func searchHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	query := r.PostForm["search"][0]

	w.Write(renderWordsTable(query))
	log.Printf("Rendered a table for query %s", query)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("./templates/index.html"))
	user := User{"Svu"}
	t.Execute(w, user)
	log.Printf("Hi, %s", user.Name)
}

func main() {
	port := ":8080"

	router := http.NewServeMux()
	database, _ := sql.Open("sqlite3", "./words.db")

	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS Words ( Woord TEXT PRIMARY KEY, Woordsoort TEXT, Uitspraak TEXT, Vertaling TEXT );")
	statement.Exec()

	fs := http.FileServer(http.Dir("./static"))
	router.Handle("GET /static/", http.StripPrefix("/static/", fs))

	router.HandleFunc("GET /", indexHandler)
	router.HandleFunc("POST /", searchHandler)

	server := http.Server{
		Addr:    port,
		Handler: router,
	}

	log.Printf("Starting server http://localhost%s", server.Addr)
	server.ListenAndServe()
}
