package tory

import (
	"fmt"
	"log"
	"os"

	"github.com/Sirupsen/logrus"
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
				ip inet NOT NULL,
				tags hstore,
				vars hstore,
				modified timestamp DEFAULT current_timestamp
			)`,
			`CREATE INDEX hosts_package_idx ON hosts (package)`,
			`CREATE INDEX hosts_image_idx ON hosts (image)`,
			`CREATE INDEX hosts_type_idx ON hosts (type)`,
			`CREATE INDEX hosts_ip_idx ON hosts (ip)`,
			`CREATE INDEX hosts_tags_idx ON hosts USING GIN (tags)`,
			`CREATE INDEX hosts_vars_idx ON hosts USING GIN (vars)`,
		},
	}

	missingHostError = fmt.Errorf("no such host")
)

func init() {
	if DefaultDatabaseURL == "" {
		DefaultDatabaseURL = "postgres://localhost/tory"
	}
}

type database struct {
	conn *sqlx.DB
	l    *log.Logger
	log  *logrus.Logger

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
		log:        logrus.New(),
	}

	if db.Migrations == nil {
		db.Migrations = defaultMigrations
	}

	return db, nil
}

func (db *database) CreateHost(h *host) error {
	tx, err := db.conn.Beginx()
	if err != nil {
		return err
	}

	rows, err := tx.NamedQuery(`
		INSERT INTO hosts (name, package, image, type, ip, tags, vars) 
		VALUES (:name, :package, :image, :type, :ip, :tags, :vars)
		RETURNING id`, h)
	if err != nil {
		defer tx.Rollback()
		return err
	}

	for rows.Next() {
		err = rows.StructScan(h)
		if err != nil {
			db.log.WithFields(logrus.Fields{"err": err}).Error("failed to scan struct")
			return tx.Rollback()
		}
	}

	db.log.WithFields(logrus.Fields{"host": h}).Info("created host")
	return tx.Commit()
}

func (db *database) ReadHost(identifier string) (*host, error) {
	row := db.conn.QueryRowx(`SELECT * FROM hosts WHERE name = $1 OR ip::text = $1`, identifier)
	if row == nil {
		return nil, missingHostError
	}

	h := newHost()
	err := row.StructScan(h)
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (db *database) ReadAllHosts() ([]*host, error) {
	rows, err := db.conn.Queryx(`SELECT * FROM hosts`)
	if err != nil {
		return nil, err
	}

	hosts := []*host{}
	count := 0
	for rows.Next() {
		h := newHost()
		err = rows.StructScan(h)
		if err != nil {
			db.log.WithFields(logrus.Fields{"err": err}).Error("failed to scan struct")
			return nil, err
		}
		hosts = append(hosts, h)
		count++
	}

	db.log.WithFields(logrus.Fields{"count": count}).Info("returning all hosts")
	return hosts, nil
}

func (db *database) UpdateHost(h *host) error {
	return nil
}

func (db *database) DeleteHost(name, ip string) error {
	return nil
}

func (db *database) Setup() error {
	ensurer := sensurer.New(db.conn.DB, db.Migrations, db.l)
	return ensurer.EnsureSchema()
}
