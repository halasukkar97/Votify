package database

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// DB is the shared database connection used by handlers that need PostgreSQL.
// It is set once by Connect when the application starts.
var DB *sql.DB

// Connect opens a PostgreSQL connection and verifies it with Ping.
// sql.Open prepares the connection, while Ping confirms the database is reachable.
func Connect(connectionString string) error {
	var err error

	DB, err = sql.Open(
		"postgres",
		connectionString,
	)

	if err != nil {
		return err
	}

	return DB.Ping()
}
