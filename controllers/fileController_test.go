package controllers

import (
	"authentication/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestUploadFileHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	config.DB = db
	defer db.Close()

	mock.ExpectExec("INSERT INTO files").
		WithArgs("user_id", "file_name", "file_size", "file_url", "file_extension", false, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(`{"file_name":"testfile.txt","file_size":1234}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	UploadFileHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
