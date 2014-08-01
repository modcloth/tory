package tory

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/modcloth-labs/schema_ensurer"
	// register the pq stuff
	_ "github.com/lib/pq"
)

var (
	// DefaultDatabaseURL is the default value for connecting to the database
	DefaultDatabaseURL = os.Getenv("DATABASE_URL")

	defaultMigrations = map[string][]string{
		"2014-08-01T19:18:12": []string{
			`CREATE EXTENSION IF NOT EXISTS hstore`,
			`CREATE SEQUENCE hosts_serial`,
			`CREATE TABLE IF NOT EXISTS hosts (
				id integer PRIMARY KEY DEFAULT nextval('hosts_serial'),
				name varchar(255) UNIQUE NOT NULL,
				package varchar(255),
				image varchar(255),
				type varchar(255),
				ip varchar(15) NOT NULL,
				tags hstore,
				attrs hstore,
				modified timestamp DEFAULT current_timestamp
			)`,
			`CREATE INDEX hosts_package_idx ON hosts (package)`,
			`CREATE INDEX hosts_image_idx ON hosts (image)`,
			`CREATE INDEX hosts_type_idx ON hosts (type)`,
			`CREATE INDEX hosts_ip_idx ON hosts (ip)`,
			`CREATE INDEX hosts_tags_idx ON hosts USING GIN (tags)`,
			`CREATE INDEX hosts_attrs_idx ON hosts USING GIN (attrs)`,
		},
	}
)

func init() {
	if DefaultDatabaseURL == "" {
		DefaultDatabaseURL = "postgres://localhost/tory"
	}
}

type database struct {
	conn *sqlx.DB
	l    *log.Logger

	Migrations map[string][]string
}

func newDatabase(urlString string, migrations map[string][]string) (*database, error) {
	conn, err := sqlx.Connect("postgres", urlString)
	if err != nil {
		return nil, err
	}

	db := &database{
		conn:       conn,
		Migrations: migrations,
		l:          log.New(os.Stderr, "", log.LstdFlags),
	}

	if db.Migrations == nil {
		db.Migrations = defaultMigrations
	}

	return db, nil
}

func (db *database) CreateHost(host *host) error {
	return nil
}

func (db *database) ReadHost(name, ip string) (*host, error) {
	return nil, nil
}

func (db *database) UpdateHost(host *host) error {
	return nil
}

func (db *database) DeleteHost(name, ip string) error {
	return nil
}

func (db *database) Setup() error {
	ensurer := sensurer.New(db.conn.DB, db.Migrations, db.l)
	return ensurer.EnsureSchema()
}
