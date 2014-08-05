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
	IP   *inet  `db:"ip"`

	Package sql.NullString `db:"package"`
	Image   sql.NullString `db:"image"`
	Type    sql.NullString `db:"type"`

	Tags *hstore.Hstore `db:"tags"`
	Vars *hstore.Hstore `db:"vars"`

	Modified time.Time `db:"modified"`
}

type HostJSON struct {
	ID int64 `json:"id,omitempty"`

	Name string `json:"name"`
	IP   string `json:"ip"`

	Package string `json:"package,omitempty"`
	Image   string `json:"image,omitempty"`
	Type    string `json:"type,omitempty"`

	Tags map[string]interface{} `json:"tags,omitempty"`
	Vars map[string]interface{} `json:"vars,omitempty"`
}

type HostPayload struct {
	Host *HostJSON `json:"host"`
}

func newHost() *host {
	return &host{
		Tags: &hstore.Hstore{},
		Vars: &hstore.Hstore{},
	}
}

func NewHostJSON() *HostJSON {
	return &HostJSON{
		Tags: map[string]interface{}{},
		Vars: map[string]interface{}{},
	}
}

func hostJSONToHost(hj *HostJSON) *host {
	h := &host{
		ID:      hj.ID,
		Name:    hj.Name,
		IP:      &inet{Addr: hj.IP},
		Package: sql.NullString{String: hj.Package, Valid: true},
		Image:   sql.NullString{String: hj.Image, Valid: true},
		Type:    sql.NullString{String: hj.Type, Valid: true},
		Tags:    &hstore.Hstore{Map: map[string]sql.NullString{}},
		Vars:    &hstore.Hstore{Map: map[string]sql.NullString{}},
	}

	for key, value := range hj.Tags {
		h.Tags.Map[fmt.Sprintf("%s", key)] = sql.NullString{
			String: fmt.Sprintf("%s", value),
			Valid:  true,
		}
	}

	for key, value := range hj.Vars {
		h.Vars.Map[fmt.Sprintf("%s", key)] = sql.NullString{
			String: fmt.Sprintf("%s", value),
			Valid:  true,
		}
	}

	return h
}

func hostToHostJSON(h *host) *HostJSON {
	hj := &HostJSON{
		ID:      h.ID,
		Name:    h.Name,
		IP:      h.IP.Addr,
		Package: h.Package.String,
		Image:   h.Image.String,
		Type:    h.Type.String,
		Tags:    map[string]interface{}{},
		Vars:    map[string]interface{}{},
	}

	for key, value := range h.Tags.Map {
		hj.Tags[fmt.Sprintf("%s", key)] = value.String
	}

	for key, value := range h.Vars.Map {
		hj.Vars[fmt.Sprintf("%s", key)] = value.String
	}

	return hj
}

func (h *host) CollapsedVars() map[string]string {
	varsMap := map[string]string{}

	for key, value := range h.Tags.Map {
		varsMap[key] = value.String
	}
	for key, value := range h.Vars.Map {
		varsMap[key] = value.String
	}

	return varsMap
}
