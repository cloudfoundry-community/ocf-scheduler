package postgres

import (
	"database/sql"
	"os"

	"gopkg.in/khaiql/dbcleaner.v2"
	"gopkg.in/khaiql/dbcleaner.v2/engine"

	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/starkandwayne/scheduler-for-ocf/postgres/migrations"
)

var Cleaner = dbcleaner.New()
var testdb *sql.DB

func init() {
	dbUrl := os.Getenv("DATABASE_URL")

	postgres := engine.NewPostgresEngine(dbUrl)
	Cleaner.SetEngine(postgres)

	testdb, _ = sql.Open("postgres", dbUrl)

	migrate.Exec(testdb, "postgres", migrations.Collection, migrate.Up)
}
