package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func setupTestDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(DatabaseSchema)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func TestHighlightQuery(t *testing.T) {
	text := "Hello world"
	query := "world"
	expected := "Hello <b>world</b>"
	result := highlightQuery(text, query)
	assert.Equal(t, expected, result)

	query = "foo"
	expected = "Hello world"
	result = highlightQuery(text, query)
	assert.Equal(t, expected, result)
}

func TestNewTableTmplData(t *testing.T) {
	words := []Word{
		{Woord: "word1", Woordsoort: "noun", Uitspraak: "word1", Vertaling: "word1"},
		{Woord: "word2", Woordsoort: "verb", Uitspraak: "word2", Vertaling: "word2"},
	}
	data := NewTableTmplData(&words, 10)
	assert.Equal(t, 2, data.Count.Matched)
	assert.Equal(t, 10, data.Count.Total)
}

func TestRenderWordsTable(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	c := &Context{db: db}

	// Insert mock data
	db.Exec("INSERT INTO users (username, hashed_password) VALUES (?, ?)", "user1", "hashed_password")
	db.Exec("INSERT INTO words (user_id, word, word_type, pronunciation, translation) VALUES (1, 'hello', 'noun', 'hello', 'hallo')")

	search := "hello"
	userID := 1
	result := c.renderWordsTable(search, userID)

	assert.Contains(t, string(result), "<b>hello</b>")
	assert.Contains(t, string(result), "hallo")
}

func TestSearchHandler(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	c := &Context{db: db}

	// Insert mock data
	db.Exec("INSERT INTO users (username, hashed_password) VALUES (?, ?)", "user1", "hashed_password")
	db.Exec("INSERT INTO session_keys (user_id, session_key) VALUES (1, 'valid_session_key')")

	req, _ := http.NewRequest("POST", "/search", strings.NewReader("search=hello"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_key", Value: "valid_session_key"})
	rr := httptest.NewRecorder()

	c.search(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAddHandler(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	c := &Context{db: db}

	// Insert mock data
	db.Exec("INSERT INTO users (username, hashed_password) VALUES (?, ?)", "user1", "hashed_password")
	db.Exec("INSERT INTO session_keys (user_id, session_key) VALUES (1, 'valid_session_key')")

	form := "woord=testword&woordsoort=noun&uitspraak=test&vertaling=testtranslation"
	req, _ := http.NewRequest("POST", "/add", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_key", Value: "valid_session_key"})
	rr := httptest.NewRecorder()

	c.add(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDeleteHandler(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	c := &Context{db: db}

	// Insert mock data
	db.Exec("INSERT INTO users (username, hashed_password) VALUES (?, ?)", "user1", "hashed_password")
	db.Exec("INSERT INTO session_keys (user_id, session_key) VALUES (1, 'valid_session_key')")
	db.Exec("INSERT INTO words (user_id, word, word_type, pronunciation, translation) VALUES (1, 'deleteword', 'noun', 'deleteword', 'deleteword')")

	req, _ := http.NewRequest("POST", "/delete/deleteword", nil)
	req.AddCookie(&http.Cookie{Name: "session_key", Value: "valid_session_key"})
	rr := httptest.NewRecorder()

	c.delete(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
