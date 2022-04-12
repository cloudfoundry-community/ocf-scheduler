package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "1648267008",

		Up: []string{
			`CREATE TABLE executions (
			  guid CHAR(36) PRIMARY KEY,
				ref_guid CHAR(36) NOT NULL,
				ref_type TEXT NOT NULL,
				task_guid CHAR(36) DEFAULT NULL,
				schedule_guid CHAR(36) DEFAULT NULL,
				scheduled_time TIMESTAMP WITH TIME ZONE DEFAULT NULL,
				message TEXT,
				state TEXT,
				execution_start_time TIMESTAMP WITH TIME ZONE DEFAULT NULL,
				execution_end_time TIMESTAMP WITH TIME ZONE DEFAULT NULL
			);`,
		},

		Down: []string{`DROP TABLE IF EXISTS executions;`},
	}

	Collection.Migrations = append(Collection.Migrations, m)
}
