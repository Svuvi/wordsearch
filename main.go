package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "net/http/pprof"

	_ "github.com/mattn/go-sqlite3"
)

type Context struct {
	db *sql.DB
}

var DatabaseSchema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	hashed_password TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS words (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	word TEXT NOT NULL,
	word_type TEXT,
	pronunciation TEXT,
	translation TEXT,
	FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE TABLE IF NOT EXISTS session_keys (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	session_key TEXT NOT NULL UNIQUE,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(id)
);`

func main() {
	port := ":8080"
	router := http.NewServeMux()

	db, err := sql.Open("sqlite3", "./words.db")
	if err != nil {
		log.Fatal(err)
	}

	db.Exec(DatabaseSchema)

	c := Context{
		db: db,
	}

	fs := http.FileServer(http.Dir("./static"))
	router.Handle("GET /static/", http.StripPrefix("/static/", fs))

	router.HandleFunc("GET /", c.indexPage)
	router.HandleFunc("POST /", c.search)
	router.HandleFunc("POST /add/", c.add)
	router.HandleFunc("DELETE /delete/{woord}", c.delete)
	router.HandleFunc("DELETE /delete/", c.delete) // a way to delete an empty string word
	router.HandleFunc("GET /login", c.loginPage)
	router.HandleFunc("POST /login", c.loginForm)
	router.HandleFunc("GET /logout", c.logout)

	wrappedRouter := NewLogger(router)

	server := http.Server{
		Addr:    port,
		Handler: wrappedRouter,
	}

	log.Printf("Starting server http://localhost%s", server.Addr)
	server.ListenAndServe()
}
