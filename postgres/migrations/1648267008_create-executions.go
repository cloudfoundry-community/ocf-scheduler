package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "1648267008",

		Up: []string{
			`CREATE TABLE executions (
			  guid CHAR(36) PRIMARY KEY,
				ref_guid CHAR(36),
				ref_type TEXT,
				task_guid CHAR(36),
				schedule_guid CHAR(36),
				scheduled_time TIMESTAMP WITH TIME ZONE,
				message TEXT,
				state TEXT,
				execution_start_time TIMESTAMP WITH TIME ZONE NOT NULL,
				execution_end_time TIMESTAMP WITH TIME ZONE NOT NULL
			);`,
		},

		Down: []string{`DROP TABLE IF EXISTS executions;`},
	}

	Collection.Migrations = append(Collection.Migrations, m)
}
