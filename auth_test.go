package main

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func insertTestData(db *sql.DB) error {
	_, err := db.Exec(`INSERT INTO users (username, hashed_password) VALUES ('testuser', 'hashedpassword')`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO session_keys (user_id, session_key) VALUES (1, 'testsessionkey')`)
	return err
}

func TestIsAutorised(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	err = insertTestData(db)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	ctx := &Context{db: db}

	tests := []struct {
		name           string
		cookie         *http.Cookie
		expectedUser   string
		expectedStatus bool
		expectedID     int
	}{
		{
			name:           "Valid session key",
			cookie:         &http.Cookie{Name: "session_key", Value: "testsessionkey"},
			expectedUser:   "testuser",
			expectedStatus: true,
			expectedID:     1,
		},
		{
			name:           "Invalid session key",
			cookie:         &http.Cookie{Name: "session_key", Value: "invalidsessionkey"},
			expectedUser:   "",
			expectedStatus: false,
			expectedID:     0,
		},
		{
			name:           "No session key",
			cookie:         nil,
			expectedUser:   "",
			expectedStatus: false,
			expectedID:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			username, status, id := ctx.isAutorised(req)
			if username != tt.expectedUser || status != tt.expectedStatus || id != tt.expectedID {
				t.Errorf("isAutorised() = (%v, %v, %v), want (%v, %v, %v)", username, status, id, tt.expectedUser, tt.expectedStatus, tt.expectedID)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	err = insertTestData(db)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	ctx := &Context{db: db}

	tests := []struct {
		name      string
		cookie    *http.Cookie
		expectErr bool
	}{
		{
			name:      "Valid session key",
			cookie:    &http.Cookie{Name: "session_key", Value: "testsessionkey"},
			expectErr: false,
		},
		{
			name:      "Invalid session key",
			cookie:    &http.Cookie{Name: "session_key", Value: "invalidsessionkey"},
			expectErr: true,
		},
		{
			name:      "No session key",
			cookie:    nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			rr := httptest.NewRecorder()

			ctx.logout(rr, req)

			if !tt.expectErr {
				// Check if the session key is deleted from the database
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM session_keys WHERE session_key = ?", tt.cookie.Value).Scan(&count)
				if err != nil {
					t.Fatalf("Failed to query session_keys: %v", err)
				}
				if count != 0 {
					t.Errorf("Session key was not deleted")
				}

				// Check if the cookie is cleared
				clearedCookie := rr.Result().Cookies()
				if len(clearedCookie) == 0 {
					t.Fatalf("Expected cleared cookie, but got none")
				}
				if clearedCookie[0].Value != "" || clearedCookie[0].MaxAge != -1 {
					t.Errorf("Cookie was not properly cleared")
				}
			} else {
				// Check if the session key remains in the database for invalid session key
				if tt.cookie != nil {
					var count int
					err = db.QueryRow("SELECT COUNT(*) FROM session_keys WHERE session_key = ?", tt.cookie.Value).Scan(&count)
					if err != nil {
						t.Fatalf("Failed to query session_keys: %v", err)
					}
					if count != 1 && tt.cookie.Value == "testsessionkey" {
						t.Errorf("Session key should not have been deleted for invalid session")
					}
				}
			}
		})
	}
}

func insertTestData2(db *sql.DB, username, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO users (username, hashed_password) VALUES (?, ?)`, username, string(hashedPassword))
	return err
}

func TestLoginForm(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	ctx := &Context{db: db}

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Empty username",
			username:       "",
			password:       "password",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "<p>You can't use empty username!</p>",
		},
		{
			name:           "Empty password",
			username:       "user",
			password:       "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "<p>Password can't be empty!</p>",
		},
		{
			name:           "Valid username",
			username:       "testuser",
			password:       "password",
			expectedStatus: http.StatusOK,
			expectedBody:   "<p>Logged in!</p>",
		},
		{
			name:           "Invalid password",
			username:       "testuser",
			password:       "wrongpassword",
			expectedStatus: http.StatusOK,
			expectedBody:   "<p>User testuser esists, but the password doesn't match</p>",
		},
		{
			name:           "New user registration",
			username:       "newuser",
			password:       "newpassword",
			expectedStatus: http.StatusOK,
			expectedBody:   "<p>Registration successful! Make sure to remember your password, because there is no way to restore it</p>",
		},
	}

	// Insert a user for the login tests
	err = insertTestData2(db, "testuser", "password")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formData := "username=" + tt.username + "&password=" + tt.password
			req := httptest.NewRequest("POST", "http://example.com/login", bytes.NewBufferString(formData))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rr := httptest.NewRecorder()

			ctx.loginForm(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if body := rr.Body.String(); body != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, body)
			}
		})
	}
}
