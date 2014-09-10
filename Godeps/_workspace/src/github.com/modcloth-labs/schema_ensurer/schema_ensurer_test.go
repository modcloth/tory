package sensurer_test

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"testing"

	_ "code.google.com/p/gosqlite/sqlite3"
	"github.com/modcloth-labs/schema_ensurer"
)

type columnSchema struct {
	Name     string
	DataType string
	Nullable int
	Default  sql.NullString
}

var tests = []struct {
	migrations map[string][]string
	schema     map[string]columnSchema
}{
	{
		map[string][]string{
			"20120505000000": {`
	  CREATE TABLE IF NOT EXISTS some_table(
      some_number INTEGER
	  );
	  `,
				`ALTER TABLE some_table ADD COLUMN some_text TEXT`,
			},
			"20130725000000": {
				`ALTER TABLE some_table ADD COLUMN some_real REAL`,
			},
		},
		map[string]columnSchema{
			"some_real": columnSchema{
				Name:     "some_real",
				DataType: "REAL",
				Nullable: 0,
				Default:  sql.NullString{Valid: false},
			},
			"some_number": columnSchema{
				Name:     "some_number",
				DataType: "INTEGER",
				Nullable: 0,
				Default:  sql.NullString{Valid: false},
			},
			"some_text": columnSchema{
				Name:     "some_text",
				DataType: "TEXT",
				Nullable: 0,
				Default:  sql.NullString{Valid: false},
			},
		},
	},
}

var nullLogger = log.New(ioutil.Discard, "", 0)

func tableSchema(db *sql.DB, table string) (columnSchemas map[string]columnSchema, err error) {
	var ignore interface{}

	columnSchemas = make(map[string]columnSchema)

	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		p := columnSchema{}

		if err := rows.Scan(&ignore, &p.Name, &p.DataType, &p.Nullable, &p.Default, &ignore); err != nil {
			return nil, err
		}

		columnSchemas[p.Name] = p
	}

	return columnSchemas, nil
}

func TestEnsureSchema(t *testing.T) {
	var (
		db              *sql.DB
		generatedSchema map[string]columnSchema
		err             error
	)

	for i, tt := range tests {
		if db, err = sql.Open("sqlite3", ":memory:"); err != nil {
			t.Fatal(err)
		}

		schemaEnsurer := sensurer.New(db, tt.migrations, nullLogger)

		schemaEnsurer.EnsureSchema()

		if generatedSchema, err = tableSchema(db, "some_table"); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(generatedSchema, tt.schema) {
			t.Errorf("%d. Expected to get %+v, but got %+v\n", i, tt.schema, generatedSchema)
		}
	}
}
