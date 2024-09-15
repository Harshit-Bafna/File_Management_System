package controllers

import (
	"authentication/config"
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	var err error
	config.DB, err = sql.Open("postgres", "user=test dbname=test sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	code := m.Run()

	config.DB.Close()
	os.Exit(code)
}

func TestRegisterHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	config.DB = db
	defer db.Close()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"test@example.com","password":"hashed_password"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	RegisterHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
