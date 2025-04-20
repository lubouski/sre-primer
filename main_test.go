package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func init() {
	// Initialize the global logger to prevent nil dereference
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func TestGetUserHandler(t *testing.T) {
	// Setup mock DB
	var mock sqlmock.Sqlmock
	var mockDB *sql.DB
	var err error

	mockDB, mock, err = sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock db: %s", err)
	}
	defer mockDB.Close()
	db = mockDB
	dbTableName = "users"

	username := "Alice"
	// Date format as expected by `time.Parse("2006-01-02T15:04:05Z", dateOfBirth)`
	dob := time.Date(1990, 4, 20, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)

	mock.ExpectQuery("SELECT date_of_birth FROM users WHERE username = \\$1").
		WithArgs(username).
		WillReturnRows(sqlmock.NewRows([]string{"date_of_birth"}).AddRow(dob))

	// Build request
	req := httptest.NewRequest(http.MethodGet, "/hello/"+username, nil)
	req = req.WithContext(context.Background())
	req.SetPathValue("username", username)

	rec := httptest.NewRecorder()
	getUserHandler(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var response map[string]string
	err = json.NewDecoder(rec.Body).Decode(&response)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(response["message"], "Hello, Alice!") {
		t.Errorf("unexpected message: %s", response["message"])
	}
}

func TestPutUserHandler(t *testing.T) {
	// Setup mock DB
	var mock sqlmock.Sqlmock
	var mockDB *sql.DB
	var err error

	mockDB, mock, err = sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock db: %s", err)
	}
	defer mockDB.Close()
	db = mockDB
	dbTableName = "users"

	// Init logger
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	username := "Alice"
	dob := "1990-04-20"

	// Expect INSERT with UPSERT
	mock.ExpectExec(`INSERT INTO users \(username, date_of_birth\) .* ON CONFLICT`).
		WithArgs(username, dob).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Create request with valid body
	body := strings.NewReader(`{"dateOfBirth":"` + dob + `"}`)
	req := httptest.NewRequest(http.MethodPut, "/hello/"+username, body)
	req = req.WithContext(context.Background())
	req.SetPathValue("username", username)

	// Record response
	rec := httptest.NewRecorder()
	putUserHandler(rec, req)

	// Assert response
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}
}

