package migrations

import (
	migrate "github.com/rubenv/sql-migrate"
)

func init() {
	m := &migrate.Migration{
		Id: "1648243100",

		Up: []string{
			`CREATE TABLE jobs (
			  guid CHAR(36) PRIMARY KEY,
				name TEXT NOT NULL,
				command TEXT NOT NULL,
				disk_in_mb INT NOT NULL DEFAULT 1024,
				memory_in_mb INT NOT NULL DEFAULT 1024,
				state TEXT NOT NULL,
				app_guid CHAR(36) NOT NULL,
				space_guid CHAR(36) NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL
			);`,
			`CREATE UNIQUE INDEX idx_jobs_name_app_guid ON jobs(name, app_guid);`,
		},

		Down: []string{`DROP TABLE IF EXISTS jobs;`},
	}

	Collection.Migrations = append(Collection.Migrations, m)
}
