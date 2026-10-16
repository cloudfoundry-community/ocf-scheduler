# Connecting to the scheduler's database

**Requires:** `psql` access to the scheduler's database. For a scheduler
deployed with the BOSH release and its colocated PostgreSQL, that means
BOSH director access to `bosh ssh` onto the scheduler VM.

Some procedures in this guide read or change the scheduler's database
directly, such as rolling back across a release that added migrations.
This page says where the database is and how to reach it.

## Where the scheduler keeps its data

The scheduler keeps everything in one PostgreSQL database, the one that
its `DATABASE_URL` setting names. The scheduler creates and migrates its
tables itself when it starts.

| Table | Holds |
|---|---|
| `jobs` | Jobs |
| `calls` | Calls, including each call's stored authorization header |
| `schedules` | Schedules, for both jobs and calls |
| `executions` | Executions, for both jobs and calls |
| `service_instances` | Service broker instances, one for each space (2.2.0 and later) |
| `service_bindings` | Service broker bindings (2.2.0 and later) |
| `gorp_migrations` | The migrations applied so far, one row each |

The `calls` table holds the authorization headers that calls send to their
endpoints, which are secrets that belong to the calls' owners. Select only
the columns a procedure needs, and never `SELECT *` from `calls`.

## Reading safely

Read-only queries are safe while the scheduler runs. Anything that writes,
such as a rollback, starts by stopping every scheduler instance, because a
running scheduler keeps its schedules in memory and writes executions as
they run. Each procedure that writes says so in its steps.

Never pass the database password on the command line, such as
`psql postgres://user:password@host/db`. The command line can be read
from the process list and lands in shell history. Type it at `psql`'s
prompt, or put it in a `~/.pgpass` file that only you can read.

## Connecting to the database

Connect with `psql`, using the host, port, database, and user that the
scheduler's `DATABASE_URL` names, and check where you are with
`\conninfo` before running anything else. How you reach the database
host depends on how the scheduler was deployed:

- **BOSH release, with colocated PostgreSQL.** The database runs on the
  scheduler VM. See *BOSH release README → Connecting to the Colocated
  Database*
  ([link](https://github.com/cloudfoundry-community/ocf-scheduler-boshrelease/blob/main/README.md#connecting-to-the-colocated-database)).
- **External PostgreSQL.** Connect to that server the way its operators
  provide.
