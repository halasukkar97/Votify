package api

import (
	"database/sql"
	"testing"
	"votify/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
)

// newMockDatabase gives API tests a fake database connection.
// The application code still uses database.DB, but sqlmock controls every query result.
func newMockDatabase(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}

	previousDB := database.DB
	database.DB = db

	t.Cleanup(func() {
		db.Close()
		database.DB = previousDB
	})

	return db, mock
}

// requireExpectations makes sure the code sent exactly the SQL the test expected.
func requireExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}
