package main

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"strings"
	"text/template"

	_ "net/http/pprof"

	_ "github.com/mattn/go-sqlite3"
)

type Handler struct {
	db *sql.DB
}

type Word struct {
	Woord      string
	Woordsoort string
	Uitspraak  string
	Vertaling  string
}

type TableTmplData struct {
	Words *[]Word
	Count struct {
		Total   int
		Matched int
	}
}

func NewTableTmplData(words *[]Word, countTotal int) TableTmplData {
	return TableTmplData{
		Words: words,
		Count: struct {
			Total   int
			Matched int
		}{
			Total:   countTotal,
			Matched: len(*words),
		},
	}
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
	var words []Word

	queryString := "%" + query + "%"
	rows, _ := db.Query("SELECT Woord, Woordsoort, Uitspraak, Vertaling FROM Words WHERE Woord LIKE ? OR Vertaling LIKE ?;", queryString, queryString)
	for rows.Next() {
		var word Word
		rows.Scan(&word.Woord, &word.Woordsoort, &word.Uitspraak, &word.Vertaling)

		word.Woord = highlightQuery(word.Woord, query)
		word.Vertaling = highlightQuery(word.Vertaling, query)

		words = append(words, word)
		// log.Printf("{Woord: %s, Woordsoort: %s, Uitspraak: %s, Vertaling: %s}", word.Woord, word.Woordsoort, word.Uitspraak, word.Vertaling)
	}

	var wordsTotal int
	db.QueryRow("SELECT COUNT(*) FROM Words").Scan(&wordsTotal)

	data := NewTableTmplData(&words, wordsTotal)

	var wordsTable bytes.Buffer
	t.Execute(&wordsTable, data)
	return wordsTable.Bytes()
}

func (c Handler) search(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	query := r.PostForm["search"][0]

	w.Write(renderWordsTable(query, c.db))
	log.Printf("Rendered a table for query \"%s\"", query)
}

func (c Handler) add(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	log.Println(r.PostForm)

	if len(r.PostForm["woord"][0]) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("The user tried to add an empty string as a word")
		return
	}

	newWord := Word{Woord: r.PostForm["woord"][0],
		Woordsoort: r.PostForm["woordsoort"][0],
		Uitspraak:  r.PostForm["uitspraak"][0],
		Vertaling:  r.PostForm["vertaling"][0]}

	statement, _ := c.db.Prepare("INSERT OR REPLACE INTO Words (Woord, Woordsoort, Uitspraak, Vertaling) VALUES (?, ?, ?, ?)")
	_, err := statement.Exec(newWord.Woord, newWord.Woordsoort, newWord.Uitspraak, newWord.Vertaling)
	if err != nil {
		log.Print(err)
	}
}

func (c Handler) delete(w http.ResponseWriter, r *http.Request) {
	woord := r.PathValue("woord")
	log.Printf("Deleting word: '%s'", woord)

	var query string
	var err error
	if woord == "" {
		log.Println("Deleting empty string word")
		query = "DELETE FROM Words WHERE Woord = '' OR Woord IS NULL;" // for some reason doesnt work here, but does work in SQLite DB browser
		_, err = c.db.Exec(query)
	} else {
		query = "DELETE FROM Words WHERE Woord = $1;"
		_, err = c.db.Exec(query, woord)
	}

	if err != nil {
		log.Printf("Error when deleting word: '%s", woord)
		log.Print(err)
	}
}

func (c Handler) index(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("./templates/index.html"))
	t.Execute(w, "")
}

func main() {
	port := ":8080"
	router := http.NewServeMux()

	database, err := sql.Open("sqlite3", "./words.db")
	if err != nil {
		log.Fatal(err)
	}

	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS Words ( Woord TEXT PRIMARY KEY, Woordsoort TEXT, Uitspraak TEXT, Vertaling TEXT );")
	statement.Exec()

	handler := Handler{
		db: database,
	}

	fs := http.FileServer(http.Dir("./static"))
	router.Handle("GET /static/", http.StripPrefix("/static/", fs))

	router.HandleFunc("GET /", handler.index)
	router.HandleFunc("POST /", handler.search)
	router.HandleFunc("POST /add/", handler.add)
	router.HandleFunc("DELETE /delete/{woord}", handler.delete)
	router.HandleFunc("DELETE /delete/", handler.delete) // a way to delete an empty string word

	server := http.Server{
		Addr:    port,
		Handler: router,
	}

	log.Printf("Starting server http://localhost%s", server.Addr)
	server.ListenAndServe()
}
