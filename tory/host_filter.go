package tory

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq/hstore"
)

var (
	zeroTime time.Time
)

type hostFilter struct {
	Name   string
	Env    string
	Team   string
	Since  time.Time
	Before time.Time
}

func (hf *hostFilter) BuildWhereClause() (string, []interface{}) {
	whereParts := []string{}
	binds := []interface{}{}

	if hf.Name == "" && hf.Env == "" && hf.Team == "" && hf.Since == zeroTime && hf.Before == zeroTime {
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

	if hf.Since != zeroTime {
		binds = append(binds, hf.Since)
		whereParts = append(whereParts,
			fmt.Sprintf("modified > $%d", len(binds)))
	}

	if hf.Before != zeroTime {
		binds = append(binds, hf.Before)
		whereParts = append(whereParts,
			fmt.Sprintf("modified < $%d", len(binds)))
	}

	if len(whereParts) > 0 {
		return " WHERE " + strings.Join(whereParts, " AND "), binds
	}

	return "", binds
}
