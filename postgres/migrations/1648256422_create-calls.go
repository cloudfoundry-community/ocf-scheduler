package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "1648256422",

		Up: []string{
			`CREATE TABLE calls (
			  guid CHAR(36) PRIMARY KEY,
				name TEXT NOT NULL,
				url TEXT NOT NULL,
				auth_header TEXT NOT NULL,
				app_guid CHAR(36) NOT NULL,
				space_guid CHAR(36) NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL
			);`,
			`CREATE UNIQUE INDEX idx_calls_name_app_guid ON calls(name, app_guid);`,
		},

		Down: []string{`DROP TABLE IF EXISTS calls;`},
	}

	Collection.Migrations = append(Collection.Migrations, m)
}
