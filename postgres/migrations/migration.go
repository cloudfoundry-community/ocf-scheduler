package migrations

import (
	migrate "github.com/rubenv/sql-migrate"
)

var Collection = &migrate.MemoryMigrationSource{
	Migrations: make([]*migrate.Migration, 0),
}
