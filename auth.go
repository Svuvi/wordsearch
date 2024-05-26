package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	responseLoggedInSuccess = `<p>Logged in!</p>
	<p><a href="/">Start using Wordsearch</a></p>`
	responsePasswordNotMatch = `
	<p>User {{ . }} esists, but the password doesn't match</p>`
	responseRegistrationSuccess = `
	<p>Registration successful! Make sure to remember your password, because there is no way to restore it</p>
	<p><a href="/">Start using Wordsearch</a></p>`
	responsePswdCantBeEmpty    = `<p>Password can't be empty!</p>`
	responseUsrnameCantBeEmpty = `<p>You can't use empty username!</p>`
	responseNotTodayBro        = `<p>Not today, bro</p>`
)

// Checks session cookie in incoming request, returns if the user is authorised, their username and id in the db.
// Returns empty string, false and 0 when the request is not authorised
func (c *Context) isAutorised(r *http.Request) (username string, status bool, user_id int) {
	cookie, err := r.Cookie("session_key")
	if err == http.ErrNoCookie {
		return "", false, 0
	}

	session_key := cookie.Value
	var count int
	query := `
	SELECT 
		COUNT(*) AS session_exists, 
		u.username, 
		u.id 
	FROM 
		session_keys sk 
	JOIN 
		users u 
	ON 
		sk.user_id = u.id 
	WHERE 
		sk.session_key = ?;
	`

	c.db.QueryRow(query, session_key).Scan(&count, &username, &user_id)
	switch count {
	case 1:
		status = true
	case 0:
		status = false
		user_id = 0
	default:
		log.Fatalf("How did we get here? session: %s, username: %s, count %b", session_key, username, count)
	}
	return username, status, user_id
}

func (c *Context) loginPage(w http.ResponseWriter, r *http.Request) {
	username, _, _ := c.isAutorised(r)

	data := struct{ Username string }{Username: username}
	t := template.Must(template.ParseFiles("./templates/login-page.html"))

	t.Execute(w, data)
	log.Printf("Served login page to user: %s", username)
}

func (c *Context) loginForm(w http.ResponseWriter, r *http.Request) {
	_, status, _ := c.isAutorised(r)
	if status {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("You shouldn't be able to do it normally"))
		return
	}

	r.ParseForm()
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(responseUsrnameCantBeEmpty))
		return
	}
	if password == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(responsePswdCantBeEmpty))
		return
	}

	var count int
	var user_id int
	var hashed_password string

	err := c.db.QueryRow("SELECT COUNT(*), id, hashed_password FROM users WHERE username = ?;", username).Scan(&count, &user_id, &hashed_password)
	log.Printf("count: %d, user_id: %d, hashed_password: %s", count, user_id, hashed_password)
	if err == sql.ErrNoRows {
		log.Printf("No password for user: %s", username)
	}

	switch count {
	case 1:
		err := bcrypt.CompareHashAndPassword([]byte(hashed_password), []byte(password))
		if err != nil {
			t := template.New("t")
			t, _ = t.Parse(responsePasswordNotMatch)
			t.Execute(w, username)
			return
		}
		// Create a session and send a session cookie
		sessionKey := uuid.NewString()
		_, err = c.db.Exec("INSERT INTO session_keys (user_id, session_key) VALUES (?, ?);", user_id, sessionKey)
		if err != nil {
			log.Fatalf("Fatal failure when inserting new session key: %v", err)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "session_key",
			Value:    sessionKey,
			HttpOnly: true,
			Secure:   true,
		})
		w.Write([]byte(responseLoggedInSuccess))

	case 0:
		// Register a new user, create a session
		hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
		if err != nil {
			log.Println(err)
		}
		hashedPassword := string(hashedPasswordBytes)

		sessionKey := uuid.NewString()

		tx, err := c.db.Begin()
		if err != nil {
			log.Fatalf("Error when starting db transaction: %v", err)
		}
		insertUserStmt := "INSERT INTO users (username, hashed_password) VALUES (?, ?)"
		insertSessionStmt := "INSERT INTO session_keys (user_id, session_key) VALUES ((SELECT id FROM users WHERE username = ?), ?)"

		_, err = tx.Exec(insertUserStmt, username, hashedPassword)
		if err != nil {
			tx.Rollback()
			log.Printf("Failure when inserting user: %v", err)
		}

		_, err = tx.Exec(insertSessionStmt, username, sessionKey)
		if err != nil {
			tx.Rollback()
			log.Printf("Failure when inserting session: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			log.Printf("Failure when commiting changes made in transaction: %v", err)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_key",
			Value:    sessionKey,
			HttpOnly: true,
			Secure:   true,
		})
		w.Write([]byte(responseRegistrationSuccess))

	default:
		// Normally this shouldn't happen
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(responseNotTodayBro))
		log.Fatal("Something is wrong in this universe: ", count, username)
	}
}

func (c *Context) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_key")
	if err != nil {
		log.Println("User requested logout without being logged in to begin with")
		return
	}

	session_key := cookie.Value
	_, err = c.db.Exec("DELETE FROM session_keys WHERE session_key = ?", session_key)
	if err != nil {
		log.Printf(`Errored when deleting session_key: "%s". \n%v`, session_key, err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_key",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
