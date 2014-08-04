package tory

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq/hstore"
)

type host struct {
	ID int64 `db:"id"`

	Name string `db:"name"`
	IP   string `db:"ip"`

	Package sql.NullString `db:"package"`
	Image   sql.NullString `db:"image"`
	Type    sql.NullString `db:"type"`

	Tags  *hstore.Hstore `db:"tags"`
	Attrs *hstore.Hstore `db:"attrs"`

	Modified time.Time `db:"modified"`
}

type hostJSON struct {
	ID int64 `json:"id,omitempty"`

	Name string `json:"name"`
	IP   string `json:"ip"`

	Package string `json:"package,omitempty"`
	Image   string `json:"image,omitempty"`
	Type    string `json:"type,omitempty"`

	Tags  map[string]interface{} `json:"tags,omitempty"`
	Attrs map[string]interface{} `json:"attrs,omitempty"`
}

func newHost() *host {
	return &host{
		Tags:  &hstore.Hstore{},
		Attrs: &hstore.Hstore{},
	}
}

func newHostJSON() *hostJSON {
	return &hostJSON{
		Tags:  map[string]interface{}{},
		Attrs: map[string]interface{}{},
	}
}

func hostJSONToHost(hj *hostJSON) *host {
	h := &host{
		ID:      hj.ID,
		Name:    hj.Name,
		IP:      hj.IP,
		Package: sql.NullString{String: hj.Package, Valid: true},
		Image:   sql.NullString{String: hj.Image, Valid: true},
		Type:    sql.NullString{String: hj.Type, Valid: true},
		Tags:    &hstore.Hstore{Map: map[string]sql.NullString{}},
		Attrs:   &hstore.Hstore{Map: map[string]sql.NullString{}},
	}

	for key, value := range hj.Tags {
		h.Tags.Map[fmt.Sprintf("%s", key)] = sql.NullString{
			String: fmt.Sprintf("%s", value),
			Valid:  true,
		}
	}

	for key, value := range hj.Attrs {
		h.Attrs.Map[fmt.Sprintf("%s", key)] = sql.NullString{
			String: fmt.Sprintf("%s", value),
			Valid:  true,
		}
	}

	return h
}

func hostToHostJSON(h *host) *hostJSON {
	hj := &hostJSON{
		ID:      h.ID,
		Name:    h.Name,
		IP:      h.IP,
		Package: h.Package.String,
		Image:   h.Image.String,
		Type:    h.Type.String,
		Tags:    map[string]interface{}{},
		Attrs:   map[string]interface{}{},
	}

	for key, value := range h.Tags.Map {
		hj.Tags[fmt.Sprintf("%s", key)] = value.String
	}

	for key, value := range h.Attrs.Map {
		hj.Attrs[fmt.Sprintf("%s", key)] = value.String
	}

	return hj
}
