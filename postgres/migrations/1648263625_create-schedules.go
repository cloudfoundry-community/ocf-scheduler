package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "1648263625",

		Up: []string{
			`CREATE TABLE schedules (
			  guid CHAR(36) PRIMARY KEY,
				enabled BOOL,
				expression TEXT NOT NULL,
				expression_type TEXT NOT NULL,
				ref_guid CHAR(36) NOT NULL,
				ref_type TEXT NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL
			);`,
		},

		Down: []string{`DROP TABLE IF EXISTS schedules;`},
	}

	Collection.Migrations = append(Collection.Migrations, m)
}
