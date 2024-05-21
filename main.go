package main

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"strings"
	"text/template"

	_ "github.com/mattn/go-sqlite3"
)

type Word struct {
	Woord      string
	Woordsoort string
	Uitspraak  string
	Vertaling  string
}

func highlightQuery(text, query string) string {
	if query == "" {
		return text
	}
	// Escape special HTML characters in the query
	escapedQuery := template.HTMLEscapeString(query)
	// Replace occurrences of the query with highlighted version
	highlighted := strings.ReplaceAll(text, escapedQuery, "<b>"+escapedQuery+"</b>")
	return highlighted
}

func renderWordsTable(query string, db *sql.DB) []byte {
	t := template.Must(template.ParseFiles("./templates/table.html"))
	/* mockData := []Word{
		{Woord: "frikandel", Woordsoort: "nw", Uitspraak: "[фрикандель]", Vertaling: ""},
		{Woord: "stroopwafel", Woordsoort: "nw", Uitspraak: "[строопвафел]", Vertaling: ""},
		{Woord: "kaneelbroodje", Woordsoort: "nw", Uitspraak: "[канельброодье]", Vertaling: "булочка с корицей"},
	} */
	var realData []Word

	queryString := "%" + query + "%"
	rows, _ := db.Query("SELECT Woord, Woordsoort, Uitspraak, Vertaling FROM Words WHERE Woord LIKE ? OR Vertaling LIKE ?;", queryString, queryString)
	for rows.Next() {
		var word Word
		rows.Scan(&word.Woord, &word.Woordsoort, &word.Uitspraak, &word.Vertaling)

		word.Woord = highlightQuery(word.Woord, query)
		word.Vertaling = highlightQuery(word.Vertaling, query)

		realData = append(realData, word)
		// log.Printf("{Woord: %s, Woordsoort: %s, Uitspraak: %s, Vertaling: %s}", word.Woord, word.Woordsoort, word.Uitspraak, word.Vertaling)
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

func addHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	r.ParseForm()
	log.Println(r.PostForm)

	newWord := Word{Woord: r.PostForm["woord"][0],
		Woordsoort: r.PostForm["woordsoort"][0],
		Uitspraak:  r.PostForm["uitspraak"][0],
		Vertaling:  r.PostForm["vertaling"][0]}

	statement, _ := db.Prepare("INSERT OR REPLACE INTO Words (Woord, Woordsoort, Uitspraak, Vertaling) VALUES (?, ?, ?, ?)")
	_, err := statement.Exec(newWord.Woord, newWord.Woordsoort, newWord.Uitspraak, newWord.Vertaling)
	if err != nil {
		log.Print(err)
	}
}

func deleteHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	woord := r.PathValue("woord")
	log.Print("Deleting word: ", woord)

	_, err := db.Exec("DELETE FROM Words WHERE Woord = $1", woord)
	if err != nil {
		log.Print("Error when deleting word: ", woord)
		log.Print(err)
	}
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
	addHandlerWithDB := func(w http.ResponseWriter, r *http.Request) {
		addHandler(w, r, database)
	}
	deleteHandlerWithDB := func(w http.ResponseWriter, r *http.Request) {
		deleteHandler(w, r, database)
	}

	fs := http.FileServer(http.Dir("./static"))
	router.Handle("GET /static/", http.StripPrefix("/static/", fs))

	router.HandleFunc("GET /", indexHandler)
	router.HandleFunc("POST /", searchHandlerWithDB)
	router.HandleFunc("POST /add/", addHandlerWithDB)
	router.HandleFunc("DELETE /delete/{woord}", deleteHandlerWithDB)

	server := http.Server{
		Addr:    port,
		Handler: router,
	}

	log.Printf("Starting server http://localhost%s", server.Addr)
	server.ListenAndServe()
}
