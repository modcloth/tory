package tory

import (
	"database/sql"
	"time"

	"github.com/lib/pq/hstore"
)

type host struct {
	ID int64 `db:"id" json:"id,omitempty"`

	Name string `db:"name" json:"name"`
	IP   string `db:"ip" json:"ip"`

	Package sql.NullString `db:"package" json:"package,omitempty"`
	Image   sql.NullString `db:"image" json:"image,omitempty"`
	Type    sql.NullString `db:"type" json:"type,omitempty"`

	Tags  *hstore.Hstore `db:"tags" json:"tags,omitempty"`
	Attrs *hstore.Hstore `db:"attrs" json:"attrs,omitempty"`

	Modified time.Time `db:"modified" json:"modified,omitempty"`
}

func newHost() *host {
	return &host{
		Tags:  &hstore.Hstore{},
		Attrs: &hstore.Hstore{},
	}
}
