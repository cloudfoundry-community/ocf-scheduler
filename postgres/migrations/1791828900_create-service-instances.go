package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "1791828900",

		Up: []string{
			`CREATE TABLE service_instances (
				instance_id TEXT PRIMARY KEY,
				service_id TEXT NOT NULL,
				plan_id TEXT NOT NULL,
				org_guid TEXT,
				space_guid TEXT NOT NULL,
				context JSONB,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL
			);`,
			`CREATE UNIQUE INDEX idx_service_instances_space_guid ON service_instances(space_guid);`,
		},

		Down: []string{`DROP TABLE IF EXISTS service_instances;`},
	}

	Collection.Migrations = append(Collection.Migrations, m)
}
