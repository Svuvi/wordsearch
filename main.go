package main

import (
	"bytes"
	"database/sql"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

type Word struct {
	Woord      string
	Woordsoort string
	Uitspraak  string
	Vertaling  string
}

func renderWordsTable(query string, db *sql.DB) []byte {
	t := template.Must(template.ParseFiles("./templates/table.html"))
	/* mockData := []Word{
		{Woord: "frikandel", Woordsoort: "nw", Uitspraak: "[фрикандель]", Vertaling: ""},
		{Woord: "stroopwafel", Woordsoort: "nw", Uitspraak: "[строопвафел]", Vertaling: ""},
		{Woord: "kaneelbroodje", Woordsoort: "nw", Uitspraak: "[канельброодье]", Vertaling: "булочка с корицей"},
	} */
	var realData []Word

	rows, _ := db.Query("SELECT Woord, Woordsoort, Uitspraak, Vertaling FROM Words WHERE Woord LIKE '%' || :query || '%';", query)
	for rows.Next() {
		var word Word
		rows.Scan(&word.Woord, &word.Woordsoort, &word.Uitspraak, &word.Vertaling)
		realData = append(realData, word)
		log.Printf("{Woord: %s, Woordsoort: %s, Uitspraak: %s, Vertaling: %s}", word.Woord, word.Woordsoort, word.Uitspraak, word.Vertaling)
	}

	var wordsTable bytes.Buffer
	t.Execute(&wordsTable, realData)
	return wordsTable.Bytes()
}

func searchHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	r.ParseForm()
	query := r.PostForm["search"][0]

	w.Write(renderWordsTable(query, db))
	log.Printf("Rendered a table for query \"%s\"", query)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("./templates/index.html"))
	t.Execute(w, "")
}

func main() {
	port := ":8080"

	router := http.NewServeMux()
	database, _ := sql.Open("sqlite3", "./words.db")

	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS Words ( Woord TEXT PRIMARY KEY, Woordsoort TEXT, Uitspraak TEXT, Vertaling TEXT );")
	statement.Exec()

	searchHandlerWithDB := func(w http.ResponseWriter, r *http.Request) {
		searchHandler(w, r, database)
	}

	fs := http.FileServer(http.Dir("./static"))
	router.Handle("GET /static/", http.StripPrefix("/static/", fs))

	router.HandleFunc("GET /", indexHandler)
	router.HandleFunc("POST /", searchHandlerWithDB)

	server := http.Server{
		Addr:    port,
		Handler: router,
	}

	log.Printf("Starting server http://localhost%s", server.Addr)
	server.ListenAndServe()
}
