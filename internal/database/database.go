package database

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// DB is kept as a small compatibility hook for package-level handler wrappers in tests.
// The production server passes the returned *sql.DB through repositories instead.
var DB *sql.DB

// Connect opens a PostgreSQL connection and verifies it with Ping.
// sql.Open prepares the connection, while Ping confirms the database is reachable.
func Connect(connectionString string) (*sql.DB, error) {
	db, err := sql.Open(
		"postgres",
		connectionString,
	)

	if err != nil {
		return nil, err
	}

	DB = db

	return db, db.Ping()
}
