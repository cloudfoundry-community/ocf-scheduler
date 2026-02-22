package migrations

import (
	"github.com/rubenv/sql-migrate"
)

func init() {
	m := &migrate.Migration{
		Id: "1771720878",

		Up: []string{
		    `ALTER TABLE jobs ADD COLUMN IF NOT EXISTS log_rate_in_bps INT NOT NULL DEFAULT 0;`,
		},
		Down: []string{`ALTER TABLE jobs DROP COLUMN IF EXISTS log_rate_in_bps;`},
	}

	Collection.Migrations = append(Collection.Migrations, m)
}
