package tory

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq/hstore"
)

type hostFilter struct {
	Name string
	Env  string
	Team string
}

func (hf *hostFilter) BuildWhereClause() (string, []interface{}) {
	whereParts := []string{}
	binds := []interface{}{}

	if hf.Name == "" && hf.Env == "" && hf.Team == "" {
		return "", binds
	}

	if hf.Name != "" {
		binds = append(binds, fmt.Sprintf("%s%%", hf.Name))
		whereParts = append(whereParts, fmt.Sprintf("name like $%d", len(binds)))
	}

	if hf.Env != "" {
		binds = append(binds, hstore.Hstore{
			Map: map[string]sql.NullString{
				"env": sql.NullString{
					String: strings.ToLower(hf.Env),
					Valid:  true,
				},
			},
		})
		whereParts = append(whereParts,
			fmt.Sprintf("lower(tags::text)::hstore @> $%d", len(binds)))
	}

	if hf.Team != "" {
		binds = append(binds, hstore.Hstore{
			Map: map[string]sql.NullString{
				"team": sql.NullString{
					String: strings.ToLower(hf.Team),
					Valid:  true,
				},
			},
		})
		whereParts = append(whereParts,
			fmt.Sprintf("lower(tags::text)::hstore @> $%d", len(binds)))
	}

	if len(whereParts) > 0 {
		return " WHERE " + strings.Join(whereParts, " AND "), binds
	}

	return "", binds
}
