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
	noHostInDatabaseError = fmt.Errorf("no such host")
	createHostFailedError = fmt.Errorf("failed to create host")
	noVarError            = fmt.Errorf("no such var")
	noTagError            = fmt.Errorf("no such tag")
)

type database struct {
	conn *sqlx.DB
	l    *log.Logger
	Log  *logrus.Logger
}

type idRow struct {
	ID int `db:"id"`
}

type valueRow struct {
	Value sql.NullString `db:"value"`
}

func newDatabase(urlString string) (*database, error) {
	conn, err := sqlx.Connect("postgres", urlString)
	if err != nil {
		return nil, err
	}

	db := &database{
		conn: conn,
		l:    log.New(os.Stderr, "", log.LstdFlags),
		Log:  logrus.New(),
	}

	return db, nil
}

func (db *database) CreateHost(h *host) (*host, error) {
	tx, err := db.conn.Beginx()
	if err != nil {
		return nil, err
	}

	stmt, err := tx.PrepareNamed(`
		INSERT INTO hosts (name, package, image, type, ip, tags, vars) 
		VALUES (:name, :package, :image, :type, :ip, :tags, :vars)
		RETURNING id`)
	if err != nil {
		defer tx.Rollback()
		return nil, err
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
		return nil, err
	}

	db.Log.WithField("host", h).Info("created host")
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return db.ReadHost(h.Name)
}

func (db *database) ReadHost(identifier string) (*host, error) {
	h := newHost()
	err := db.conn.Get(h, `
		SELECT * FROM hosts
		WHERE name = $1 OR host(ip) = $1
		ORDER BY modified DESC
		LIMIT 1`, identifier)

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

func (db *database) UpdateHost(h *host) (*host, error) {
	tx, err := db.conn.Beginx()
	if err != nil {
		return nil, err
	}

	stmt, err := tx.PrepareNamed(`
		UPDATE hosts
		SET package = :package, image = :image, type = :type, ip = :ip,
			tags = tags || :tags, vars = vars || :vars,
			modified = current_timestamp
		WHERE name = :name
		RETURNING id`)

	if err != nil {
		defer tx.Rollback()
		return nil, err
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
			return nil, noHostInDatabaseError
		} else {
			db.Log.WithFields(errFields).Warn("failed to scan struct")
			return nil, err
		}
	}

	db.Log.WithField("host", h).Info("updated host")
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return db.ReadHost(h.Name)
}

func (db *database) DeleteHost(identifier string) error {
	stmt, err := db.conn.Preparex(`
		DELETE FROM hosts
		WHERE name = $1 OR host(ip) = $1
		RETURNING id`)
	if err != nil {
		return err
	}

	one := &idRow{}
	err = stmt.Get(one, identifier)
	if err != nil && err == sql.ErrNoRows {
		return noHostInDatabaseError
	}

	return err
}

func (db *database) ReadVarOrTag(which, identifier, key string) (string, error) {
	stmt, err := db.conn.Preparex(fmt.Sprintf(`
		SELECT %s -> $2 AS value
		FROM hosts
		WHERE name = $1 OR host(ip) = $1
		ORDER BY modified DESC
		LIMIT 1`, which))
	if err != nil {
		return "", err
	}

	v := &valueRow{Value: sql.NullString{}}
	err = stmt.Get(v, identifier, key)
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

func (db *database) UpdateVarOrTag(which, identifier, key, value string) error {
	stmt, err := db.conn.Preparex(fmt.Sprintf(`
		UPDATE hosts
		SET %s = %s || $2,
			modified = current_timestamp
		WHERE name = $1 OR host(ip) = $1
		RETURNING id`,
		which, which))

	if err != nil {
		return err
	}

	id := &idRow{}
	err = stmt.Get(id, identifier, &hstore.Hstore{
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

func (db *database) DeleteVarOrTag(which, identifier, key string) error {
	stmt, err := db.conn.Preparex(fmt.Sprintf(`
		UPDATE hosts
		SET %s = delete(%s, $2),
			modified = current_timestamp
		WHERE name = $1 OR host(ip) = $1
		RETURNING id`,
		which, which))

	if err != nil {
		return err
	}

	id := &idRow{}
	err = stmt.Get(id, identifier, key)
	if err != nil && err == sql.ErrNoRows {
		return noHostInDatabaseError
	}

	return err
}

func (db *database) ReadVar(name, key string) (string, error) {
	return db.ReadVarOrTag("vars", name, key)
}

func (db *database) UpdateVar(identifier, key, value string) error {
	return db.UpdateVarOrTag("vars", identifier, key, value)
}

func (db *database) DeleteVar(identifier, key string) error {
	return db.DeleteVarOrTag("vars", identifier, key)
}

func (db *database) ReadTag(name, key string) (string, error) {
	return db.ReadVarOrTag("tags", name, key)
}

func (db *database) UpdateTag(identifier, key, value string) error {
	return db.UpdateVarOrTag("tags", identifier, key, value)
}

func (db *database) DeleteTag(identifier, key string) error {
	return db.DeleteVarOrTag("tags", identifier, key)
}

func (db *database) Setup(migrations map[string][]string) error {
	ensurer := sensurer.New(db.conn.DB, migrations, db.l)
	return ensurer.EnsureSchema()
}
