package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "1791828901",

		Up: []string{
			`CREATE TABLE service_bindings (
				binding_id TEXT PRIMARY KEY,
				instance_id TEXT NOT NULL REFERENCES service_instances(instance_id),
				app_guid TEXT,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL
			);`,
			`CREATE INDEX idx_service_bindings_app_guid ON service_bindings(app_guid);`,
			`CREATE INDEX idx_service_bindings_instance_id ON service_bindings(instance_id);`,
		},

		Down: []string{`DROP TABLE IF EXISTS service_bindings;`},
	}

	Collection.Migrations = append(Collection.Migrations, m)
}
