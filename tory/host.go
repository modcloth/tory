package tory

import (
	"database/sql"

	"github.com/lib/pq/hstore"
)

type host struct {
	Name string `db:"name"`
	IP   string `db:"ip"`

	Package sql.NullString `db:"package"`
	Image   sql.NullString `db:"image"`
	Type    sql.NullString `db:"type"`

	Tags  *hstore.Hstore `db:"tags"`
	Attrs *hstore.Hstore `db:"attrs"`
}
