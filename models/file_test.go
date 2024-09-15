package models

import (
	"authentication/config"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestSaveFileMetadata(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	config.DB = db
	defer db.Close()

	mock.ExpectQuery("INSERT INTO files").
		WithArgs(1, "testfile.txt", 1234, "s3://bucket/testfile.txt", "txt", false, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	fileID, err := SaveFileMetadata(1, "testfile.txt", 1234, "s3://bucket/testfile.txt", "txt", false, time.Now())

	assert.NoError(t, err)
	assert.Equal(t, 1, fileID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
