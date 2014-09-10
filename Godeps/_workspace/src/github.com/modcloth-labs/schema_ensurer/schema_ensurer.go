// Package sensurer provides a simple interface to keep the application
// database schema up to date. It currently only assists in maintaining the
// schema to the most up-to-date version. It uses SQL99 compliant DDL and query
// language so it should work with any DBMS supporting SQL99.
package sensurer

import (
	"database/sql"
	"log"
	"sort"
)

type SchemaEnsurer struct {
	//Database connection
	DB *sql.DB

	//Keys should be labels for the migrations (will be executed in alphanumeric order)
	//Values should be an array of SQL statements
	Migrations map[string][]string

	//Logger to log debug statements to (the migrations being executed)
	Log *log.Logger
}

//New returns a new SchemaEnsurer struct initialized with the given db,
//migrations, and logger. See struct definition for argument descriptions.
func New(db *sql.DB, migrations map[string][]string, log *log.Logger) *SchemaEnsurer {
	return &SchemaEnsurer{
		DB:         db,
		Migrations: migrations,
		Log:        log,
	}
}

//EnsureSchema first creates the schema_migrations table if it does not exist
//and then determines which migrations have not been applied to the database
//and applies them.
//It returns any error that occurred while migrating.
func (me *SchemaEnsurer) EnsureSchema() error {
	if err := me.ensureMigrationsTable(); err != nil {
		return err
	}
	return me.runMigrations()
}

func (me *SchemaEnsurer) ensureMigrationsTable() error {
	_, err := me.DB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version character varying(255) NOT NULL);`)
	return err
}

func (me *SchemaEnsurer) migrationLabels() (labels []string) {
	labels = make([]string, len(me.Migrations))

	var i int
	for label, _ := range me.Migrations {
		labels[i] = label
		i++
	}

	sort.Strings(labels)

	return labels
}

func (me *SchemaEnsurer) runMigrations() error {
	for _, schemaVersion := range me.migrationLabels() {
		if me.containsMigration(schemaVersion) {
			continue
		}

		me.Log.Printf("Executing migration %s\n", schemaVersion)
		if err := me.migrateTo(schemaVersion, me.Migrations[schemaVersion]); err != nil {
			return err
		}
	}
	return nil
}

func (me *SchemaEnsurer) containsMigration(schemaVersion string) bool {
	var count int
	if err := me.DB.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", schemaVersion).Scan(&count); err != nil {
		return false
	}

	return count == 1
}

func (me *SchemaEnsurer) migrateTo(schemaVersion string, sqls []string) error {
	var (
		tx  *sql.Tx
		err error
	)

	if tx, err = me.DB.Begin(); err != nil {
		return err
	}

	for _, sql := range sqls {
		if _, err = tx.Exec(sql); err != nil {
			tx.Rollback()
			return err
		}
	}
	if _, err = tx.Exec("INSERT INTO schema_migrations VALUES ($1)", schemaVersion); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
