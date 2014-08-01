package tory

import (
	"database/sql"
	"os"

	// register the pq stuff
	_ "github.com/lib/pq"
)

var (
	// DefaultDatabaseURL is the default value for connecting to the database
	DefaultDatabaseURL = os.Getenv("DATABASE_URL")
)

func init() {
	if DefaultDatabaseURL == "" {
		DefaultDatabaseURL = "postgres://localhost/tory"
	}
}

type db struct {
	conn *sql.DB
}

func newDB(urlString string) (*db, error) {
	conn, err := sql.Open("postgres", urlString)
	if err != nil {
		return nil, err
	}

	return &db{conn: conn}, nil
}
