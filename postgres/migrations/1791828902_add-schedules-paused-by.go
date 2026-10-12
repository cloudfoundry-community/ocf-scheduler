package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "1791828902",

		Up: []string{
			`ALTER TABLE schedules ADD COLUMN IF NOT EXISTS paused_by TEXT DEFAULT NULL;`,
		},

		Down: []string{`ALTER TABLE schedules DROP COLUMN IF EXISTS paused_by;`},
	}

	Collection.Migrations = append(Collection.Migrations, m)
}
