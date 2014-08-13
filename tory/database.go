package tory

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	"github.com/modcloth-labs/schema_ensurer"
	// register the pq stuff
	_ "github.com/lib/pq"
	"github.com/lib/pq/hstore"
)

var (
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

	noHostInDatabaseError = fmt.Errorf("no such host")
	createHostFailedError = fmt.Errorf("failed to create host")
	noVarError            = fmt.Errorf("no such var")
	noTagError            = fmt.Errorf("no such tag")
)

type database struct {
	conn *sqlx.DB
	l    *log.Logger
	Log  *logrus.Logger

	Migrations map[string][]string
}

type idRow struct {
	ID int `db:"id"`
}

type valueRow struct {
	Value sql.NullString `db:"value"`
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
		Log:        logrus.New(),
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

	stmt, err := tx.PrepareNamed(`
		INSERT INTO hosts (name, package, image, type, ip, tags, vars) 
		VALUES (:name, :package, :image, :type, :ip, :tags, :vars)
		RETURNING id`)
	if err != nil {
		defer tx.Rollback()
		return err
	}

	err = stmt.Get(h, h)
	if err != nil {
		errFields := logrus.Fields{"err": err}
		if err == sql.ErrNoRows {
			db.Log.WithFields(errFields).Error("failed to create host")
		} else {
			db.Log.WithFields(errFields).Warn("failed to scan struct")
		}
		defer tx.Rollback()
		return err
	}

	db.Log.WithField("host", h).Info("created host")
	return tx.Commit()
}

func (db *database) ReadHost(identifier string) (*host, error) {
	h := newHost()
	err := db.conn.Get(h, `
		SELECT * FROM hosts
		WHERE name = $1 OR host(ip) = $1`, identifier)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, noHostInDatabaseError
		}
		return nil, err
	}

	return h, nil
}

func (db *database) ReadAllHosts(hf *hostFilter) ([]*host, error) {
	query := `SELECT * FROM hosts `
	whereClause, binds := hf.BuildWhereClause()

	query += whereClause

	db.Log.WithFields(logrus.Fields{
		"filter": hf,
		"query":  query,
		"binds":  binds,
	}).Debug("getting hosts with query and binds")

	rows, err := db.conn.Queryx(query, binds...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	hosts := []*host{}
	count := 0
	for rows.Next() {
		h := newHost()
		err = rows.StructScan(h)
		if err != nil {
			db.Log.WithField("err", err).Error("failed to scan struct")
			return nil, err
		}
		hosts = append(hosts, h)
		count++
	}

	db.Log.WithField("count", count).Info("returning all hosts")
	return hosts, nil
}

func (db *database) UpdateHost(h *host) error {
	tx, err := db.conn.Beginx()
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareNamed(`
		UPDATE hosts
		SET package = :package, image = :image, type = :type,
		    ip = :ip, tags = :tags, vars = :vars
		WHERE name = :name
		RETURNING id`)

	if err != nil {
		defer tx.Rollback()
		return err
	}

	err = stmt.Get(h, h)
	if err != nil {
		errFields := logrus.Fields{"err": err}
		defer tx.Rollback()
		if err == sql.ErrNoRows {
			// this is not considered an error because the server update is
			// doing a bit of tell-don't-ask in order to fall back to host
			// creation
			db.Log.WithFields(errFields).Warn("failed to update host")
			return noHostInDatabaseError
		} else {
			db.Log.WithFields(errFields).Warn("failed to scan struct")
			return err
		}
	}

	db.Log.WithField("host", h).Info("updated host")
	return tx.Commit()
}

func (db *database) DeleteHost(name string) error {
	stmt, err := db.conn.Preparex(`DELETE FROM hosts WHERE name = $1 RETURNING id`)
	if err != nil {
		return err
	}

	one := &idRow{}
	err = stmt.Get(one, name)
	if err != nil && err == sql.ErrNoRows {
		return noHostInDatabaseError
	}

	return err
}

func (db *database) ReadVarOrTag(which, name, key string) (string, error) {
	stmt, err := db.conn.Preparex(fmt.Sprintf(`
		SELECT %s -> $2 AS value FROM hosts WHERE name = $1`, which))
	if err != nil {
		return "", err
	}

	v := &valueRow{Value: sql.NullString{}}
	err = stmt.Get(v, name, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", noHostInDatabaseError
		}
		return "", err
	}

	if !v.Value.Valid {
		switch which {
		case "vars":
			return "", noVarError
		case "tags":
			return "", noTagError
		}
	}

	return v.Value.String, nil
}

func (db *database) UpdateVarOrTag(which, hostname, key, value string) error {
	stmt, err := db.conn.Preparex(fmt.Sprintf(`
		UPDATE hosts SET %s = %s || $2 WHERE name = $1 RETURNING id`,
		which, which))

	if err != nil {
		return err
	}

	id := &idRow{}
	err = stmt.Get(id, hostname, &hstore.Hstore{
		Map: map[string]sql.NullString{
			key: sql.NullString{
				String: value,
				Valid:  true,
			},
		},
	})

	if err != nil && err == sql.ErrNoRows {
		return noHostInDatabaseError
	}

	return err
}

func (db *database) DeleteVarOrTag(which, hostname, key string) error {
	stmt, err := db.conn.Preparex(fmt.Sprintf(`
		UPDATE hosts SET %s = delete(%s, $2) WHERE name = $1 RETURNING id`,
		which, which))

	if err != nil {
		return err
	}

	id := &idRow{}
	err = stmt.Get(id, hostname, key)
	if err != nil && err == sql.ErrNoRows {
		return noHostInDatabaseError
	}

	return err
}

func (db *database) ReadVar(name, key string) (string, error) {
	return db.ReadVarOrTag("vars", name, key)
}

func (db *database) UpdateVar(hostname, key, value string) error {
	return db.UpdateVarOrTag("vars", hostname, key, value)
}

func (db *database) DeleteVar(hostname, key string) error {
	return db.DeleteVarOrTag("vars", hostname, key)
}

func (db *database) ReadTag(name, key string) (string, error) {
	return db.ReadVarOrTag("tags", name, key)
}

func (db *database) UpdateTag(hostname, key, value string) error {
	return db.UpdateVarOrTag("tags", hostname, key, value)
}

func (db *database) DeleteTag(hostname, key string) error {
	return db.DeleteVarOrTag("tags", hostname, key)
}

func (db *database) Setup() error {
	ensurer := sensurer.New(db.conn.DB, db.Migrations, db.l)
	return ensurer.EnsureSchema()
}
