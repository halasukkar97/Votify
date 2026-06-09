package database

import (
	"database/sql"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() error {
	var err error

	DB, err = sql.Open(
		"postgres",
		"host=localhost port=5432 user=hela-sukkar dbname=movie_vote sslmode=disable",
	)

	if err != nil {
		return err
	}

	return DB.Ping()
}
