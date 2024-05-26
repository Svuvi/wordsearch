package main

import (
	"bytes"
	"log"
	"net/http"
	"strings"
	"text/template"

	_ "net/http/pprof"

	_ "github.com/mattn/go-sqlite3"
)

type Word struct {
	Woord            string
	WoordHighlighted string
	Woordsoort       string
	Uitspraak        string
	Vertaling        string
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

func (c *Context) renderWordsTable(search string, user_id int) []byte {
	t := template.Must(template.ParseFiles("./templates/table.html"))
	var words []Word

	searchString := "%" + search + "%"
	q := `
	SELECT
		word, word_type, pronunciation, translation
	FROM
		words
	WHERE
		user_id LIKE ? AND (word LIKE ? OR  translation LIKE ?);
	`
	rows, _ := c.db.Query(q, user_id, searchString, searchString)
	for rows.Next() {
		var word Word
		rows.Scan(&word.Woord, &word.Woordsoort, &word.Uitspraak, &word.Vertaling)

		word.WoordHighlighted = highlightQuery(word.Woord, search)
		word.Vertaling = highlightQuery(word.Vertaling, search)

		words = append(words, word)
		// log.Printf("{Woord: %s, Woordsoort: %s, Uitspraak: %s, Vertaling: %s}", word.Woord, word.Woordsoort, word.Uitspraak, word.Vertaling)
	}

	var wordsTotal int
	c.db.QueryRow("SELECT COUNT(*) FROM words WHERE user_id = ?", user_id).Scan(&wordsTotal)

	data := NewTableTmplData(&words, wordsTotal)

	var wordsTable bytes.Buffer
	t.Execute(&wordsTable, data)
	return wordsTable.Bytes()
}

func (c *Context) search(w http.ResponseWriter, r *http.Request) {
	_, authorised, user_id := c.isAutorised(r)
	if !authorised {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("<p>You are not autorized. <a href=\"/login\">Log in</a></p>"))
		return
	}

	r.ParseForm()
	query := r.PostForm["search"][0]

	w.Write(c.renderWordsTable(query, user_id))
	log.Printf("Rendered a table for query \"%s\"", query)
}

func (c *Context) add(w http.ResponseWriter, r *http.Request) {
	_, authorised, user_id := c.isAutorised(r)
	if !authorised {
		// w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("<p>You are not autorized. <a href=\"/login\">Log in</a></p>"))
		return
	}

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

	statement, _ := c.db.Prepare("INSERT OR REPLACE INTO words (user_id, word, word_type, pronunciation, translation) VALUES (?, ?, ?, ?, ?)")
	_, err := statement.Exec(user_id, newWord.Woord, newWord.Woordsoort, newWord.Uitspraak, newWord.Vertaling)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

func (c *Context) delete(w http.ResponseWriter, r *http.Request) {
	username, authorised, user_id := c.isAutorised(r)
	if !authorised {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("<p>You are not autorized. <a href=\"/login\">Log in</a></p>"))
		return
	}

	word := r.PathValue("woord")
	log.Printf("Deleting word: '%s', user: '%s'", word, username)

	var query string
	var err error
	if word == "" {
		log.Println("Deleting empty string word")
		query = "DELETE FROM words WHERE user_id = ? (word = ? OR word IS NULL);" // for some reason doesnt work here, but does work in SQLite DB browser
		_, err = c.db.Exec(query, user_id, word)
	} else {
		query = "DELETE FROM words WHERE user_id = ? AND word = $1;"
		_, err = c.db.Exec(query, user_id, word)
	}

	if err != nil {
		log.Printf("Error when deleting word: '%s", word)
		log.Print(err)
	}
}

func (c *Context) indexPage(w http.ResponseWriter, r *http.Request) {
	username, authorised, _ := c.isAutorised(r)
	if !authorised {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}

	data := struct{ Username string }{Username: username}
	t := template.Must(template.ParseFiles("./templates/index.html"))
	t.Execute(w, data)
}
