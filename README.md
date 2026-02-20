# OCF Scheduler

A standalone Go service that provides cron-style scheduling for Cloud Foundry applications.
It enables users to schedule CF tasks (jobs) and HTTP calls using cron expressions with timezone support.

## Features

- Schedule Cloud Foundry tasks (jobs) to run on cron expressions

- Schedule HTTP calls to run on cron expressions

- Timezone-aware scheduling with RFC 9636 timezone support

- Execution history tracking for both jobs and calls

- Configurable worker pool for concurrent execution

- Automatic cleanup of old execution records

- UAA-based authentication and authorization

## Architecture

```mermaid
graph LR
    Client([Client]) -->|REST API| API[API Server<br/>Echo]
    API -->|CRUD| PG[(PostgreSQL)]
    API -->|Add/Remove| Cron[Cron Engine]
    Cron -->|Dispatch| WP[Worker Pool]
    WP -->|Run Task| CF[CF API]
    WP -->|HTTP Request| Target[HTTP Endpoint]
    API -->|Auth| UAA[UAA]
```

On startup, the scheduler loads all enabled schedules from PostgreSQL and registers them with the cron engine.
When a schedule fires, the cron engine dispatches the work to the worker pool,
which either creates a CF task via the CF API or makes an HTTP call to the configured endpoint.

## Project Structure

| Directory | Description |
|---|---|
| `cmd/scheduler/` | Main scheduler service entry point and version metadata |
| `cmd/tzlist/` | Timezone list generation utility |
| `cmd/cli/` | CLI testing tool for interacting with the scheduler for local development |
| `cmd/jeuaa/` | Mock UAA server for local development |
| `cmd/jecfapi/` | Mock CF API server for local development |
| `core/` | Domain models (`Job`, `Call`, `Schedule`, `Execution`) and service interfaces |
| `http/` | Echo HTTP server, route handlers, and response presenters |
| `postgres/` | PostgreSQL implementations of service interfaces and database migrations |
| `cron/` | Cron scheduling engine with timezone support |
| `cf/` | Cloud Foundry integration (UAA auth, CF API info, task execution) |
| `combined/` | Combined run service that routes to job or call runners |
| `logger/` | Logrus-based logging service with custom formatter |
| `workflows/` | Business logic workflows |
| `mock/` | Mock implementations for testing |
| `features/` | Integration/feature tests |
| `scripts/` | Utility scripts (`blanket` test runner, `shait` digest generator) |
| `ci/` | Concourse CI pipeline configuration, tasks, and scripts |

## Getting Started

### Prerequisites

- Go 1.25+ (see `go.mod`)

- PostgreSQL

- Cloud Foundry deployment with UAA

- Docker and Docker Compose (optional, for local development)

### Docker Development Environment

A `docker-compose.yml` is provided for local development and testing without a real
Cloud Foundry deployment. It spins up mock UAA and CF API services alongside PostgreSQL:

```sh
docker compose up dev          # Run the scheduler with mock services
docker compose up test         # Run the test suite
```

| Service | Description | Port |
|---|---|---|
| `dev` | Scheduler service (built from `Dockerfile.dev`) | 8000 |
| `test` | Test runner (built from `Dockerfile.test`) | — |
| `postgres-dev` | PostgreSQL for development | — |
| `postgres-test` | PostgreSQL for tests | — |
| `uaa` | Mock UAA server (built from `Dockerfile.uaa`) | 8001 |
| `cf-api` | Mock CF API server (built from `Dockerfile.cfapi`) | 8002 |

The `dev` service pre-configures all required environment variables (`DATABASE_URL`,
`UAA_ENDPOINT`, `CF_ENDPOINT`, `CLIENT_ID`, `CLIENT_SECRET`) so you can start the
scheduler immediately.

### Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | | PostgreSQL connection string |
| `CF_ENDPOINT` | Yes | | Cloud Foundry API URL |
| `UAA_ENDPOINT` | Yes | | UAA server URL |
| `CLIENT_ID` | Yes | | UAA client ID |
| `CLIENT_SECRET` | Yes | | UAA client secret |
| `SCHEDULER_PORT` | No | `8000` | HTTP listen port |
| `SCHEDULER_WORKERS` | No | `20` | Worker pool size (minimum 10); values below 10 or non-numeric values log a warning and fall back to 20 |
| `LOG_LEVEL` | No | `info` | Log level (see below) |

#### Log Levels

The `LOG_LEVEL` environment variable accepts a single keyword and is parsed by
[`logrus.ParseLevel()`](https://pkg.go.dev/github.com/sirupsen/logrus#ParseLevel),
which is **case-insensitive**. The accepted values, ordered from most to least verbose:

| Value | Description |
|---|---|
| `trace` | Very fine-grained diagnostic output |
| `debug` | Debugging information |
| `info` | General operational messages (default) |
| `warn` | Potential issues that deserve attention |
| `warning` | Alias for `warn` — both are accepted |
| `error` | Errors that need investigation |
| `fatal` | Critical errors — the process will exit after logging |
| `panic` | Critical errors — the process will panic after logging |

Setting a log level enables messages at that level **and all higher-severity levels above it**.
Messages below the configured level are suppressed. For example, setting `LOG_LEVEL=warn`
will output `warn`, `error`, `fatal`, and `panic` messages, but suppress `info`, `debug`,
and `trace`.

If `LOG_LEVEL` is not set or is empty, the default is `info`.
If an invalid value is provided, a warning is logged and the level falls back to `info`.

### Building

```sh
make build        # Build for current platform
make release      # Build for all target platforms
make test         # Run tests
make clean        # Remove built executables
```

## Makefile Reference

### Targets

| Target | Description |
|---|---|
| `all` | Clean and build (default target) |
| `build` | Build for current platform; automatically adds a `dev` prerelease tag |
| `cli` | Build the CLI tool (`sch` binary) |
| `release` | Cross-compile for all target platforms and create tar.gz packages |
| `test` | Run the test suite via `scripts/blanket` |
| `clean` | Remove built executables (`./tzlist`, `./scheduler`) |
| `distclean` | Remove all release directories and packages (includes `clean`) |
| `distbuild` | Create release directory structure for all targets |
| `debug_version` | Display all computed version variables (useful for troubleshooting) |

### Public Variables

These can be overridden on the command line (e.g., `make build VERSION=1.2.3`):

| Variable | Default | Description |
|---|---|---|
| `VERSION` | *(auto from git tag + patch bump)* | Full version string (e.g., `1.2.3`, `1.2.3-alpha`) |
| `APP_NAME` | `scheduler` | Binary output name |
| `CMD_PATH` | `cmd` | Command source directory |
| `RELEASE_ROOT` | `releases` | Release output directory |
| `TARGETS` | `linux/amd64 linux/arm64 darwin/amd64 darwin/arm64` | Cross-compile target platforms |
| `MODULE` | `github.com/cloudfoundry-community/ocf-scheduler` | Go module path |
| `CGO_ENABLED` | `0` | Set to `1` to enable CGO (default produces static binaries) |
| `SEMVER_MAJOR` | *(auto)* | Major version override |
| `SEMVER_MINOR` | *(auto)* | Minor version override |
| `SEMVER_PATCH` | *(auto)* | Patch version override |
| `SEMVER_PRERELEASE` | *(auto)* | Prerelease label override |
| `SEMVER_BUILDMETA` | *(auto)* | Build metadata override |

### Private Variables

These are computed internally and not intended for direct override:

| Variable | Description |
|---|---|
| `PROJECT` | Hardcoded project name (`ocf-scheduler`) |
| `GOOS` / `GOARCH` | Detected from `go env` for the current platform |
| `GOMODULECMD` | Go package path for ldflags injection (`main`) |
| `TESTFILES` | List of Go packages (excluding vendor) |
| `CLEAN_VERSION` | `VERSION` with `v` prefix stripped |
| `HAS_BUILDMETA` / `VERSION_BUILDMETA` | Buildmeta detection and extraction from `VERSION` |
| `HAS_PRERELEASE` / `VERSION_PRERELEASE` | Prerelease detection and extraction from `VERSION` |
| `VERSION_ONLY` | Semantic version without prerelease or buildmeta |
| `VERSION_SPLIT` | Version parts as a Make word list (major minor patch) |
| `BUILD_DATE` | ISO 8601 timestamp of the build |
| `BUILD_VCS_URL` | Git remote origin URL |
| `BUILD_VCS_ID` | Short git commit hash |
| `BUILD_VCS_ID_DATE` | ISO date of the git commit |
| `GO_LDFLAGS` | Linker flags that inject version metadata into the binary |
| `SEMVER_VERSION` | Fully assembled semantic version string |

### Macros and Functions

| Name | Type | Description |
|---|---|---|
| `is_not_number` | Function | Strips digits from a string; used to validate that version components are numeric |
| `distbuild` | Macro | Creates a release directory for a given OS/ARCH combination under `RELEASE_ROOT` |
| `build-target` | Macro | Cross-compiles binaries for a specific OS/ARCH and generates SHA1/SHA256 digests via `scripts/shait` |
| `package-target` | Macro | Creates a tar.gz package for a specific OS/ARCH release directory |

### Version Auto-Detection

When `VERSION` is not specified, the Makefile:

1. Reads the latest git tag (e.g., `v1.2.3`)
2. Strips the `v` prefix
3. Validates the tag has exactly 3 numeric parts (major.minor.patch)
4. Auto-increments the patch number by 1

For example, if the latest tag is `v1.2.3`, the computed version becomes `1.2.4`.
The `build` target additionally appends `-dev` as the prerelease label (e.g., `1.2.4-dev`).

## Development Workflow

### Local Build

```sh
make build
```

Compiles `scheduler` and `tzlist` binaries for your current platform. The version is
automatically tagged with a `-dev` prerelease label (e.g., `1.2.4-dev`).

To build with a specific version:

```sh
make build VERSION=1.2.3
make build VERSION=1.2.3-beta
```

### Running Tests

```sh
make test
```

Runs the test suite via `scripts/blanket`, which supports scoped test execution:

```sh
./scripts/blanket func          # All tests, display per-function coverage
./scripts/blanket html          # All tests, open coverage report in browser
./scripts/blanket func unit     # Unit tests only
./scripts/blanket func integration  # Integration tests only
```

### Debugging Version Computation

```sh
make debug_version
```

Prints all computed version variables (`VERSION`, `CLEAN_VERSION`, `VERSION_SPLIT`,
`SEMVER_*`, etc.) — useful when troubleshooting version-related build issues.

### Cross-Compiling Locally

```sh
make release
```

Builds for all target platforms (`linux/amd64`, `linux/arm64`, `darwin/amd64`,
`darwin/arm64`), generates SHA1/SHA256 digests, and creates tar.gz packages under
the `releases/` directory.

To build a specific version as a release:

```sh
make release VERSION=1.2.3
```

### Cleaning Up

```sh
make clean       # Remove built executables (./tzlist, ./scheduler)
make distclean   # Remove all release artifacts and executables
```

## API Reference

> **Note:** The following HTTP API is an internal interface used by the scheduler service
> and its companion CLI. It is **not a public or stable API** — endpoints, request/response
> formats, and behavior may change without notice between releases. This documentation is
> provided for developer reference only.

### Authentication

All endpoints (except `GET /`) require a valid UAA bearer token in the `Authorization`
header:

```
Authorization: bearer <UAA_TOKEN>
```

Requests without a valid token receive a `401 Unauthorized` response.

### Request Bodies

**Create Job** (`POST /jobs?app_guid=`):

```json
{
  "name": "my-job",
  "command": "rake db:migrate"
}
```

Optional fields: `disk_in_mb`, `memory_in_mb`.

**Create Call** (`POST /calls?app_guid=`):

```json
{
  "name": "my-call",
  "url": "https://example.com/webhook",
  "auth_header": "bearer <token>"
}
```

**Create Schedule** (`POST /jobs/:guid/schedules` or `POST /calls/:guid/schedules`):

```json
{
  "expression": "0 30 9 * * *",
  "expression_type": "cron_expression"
}
```

The `expression` field accepts cron expressions with optional timezone support.

### Response Format

Collection endpoints return a paginated envelope:

```json
{
  "pagination": {
    "total_pages": 1,
    "total_results": 2,
    "first": { "href": "first" },
    "last": { "href": "last" },
    "next": { "href": "next" },
    "previous": { "href": "previous" }
  },
  "resources": [ ... ]
}
```

Single-resource endpoints return the entity directly as JSON.

### Jobs

| Method | Path | Description |
|---|---|---|
| POST | `/jobs?app_guid=` | Create a job |
| GET | `/jobs?space_guid=` | List all jobs in a space |
| GET | `/jobs/:guid` | Get a job |
| DELETE | `/jobs/:guid` | Delete a job |
| POST | `/jobs/:guid/execute` | Execute a job immediately |

### Job Schedules

| Method | Path | Description |
|---|---|---|
| POST | `/jobs/:guid/schedules` | Create a schedule for a job |
| GET | `/jobs/:guid/schedules` | List schedules for a job |
| DELETE | `/jobs/:guid/schedules/:schedule_guid` | Delete a job schedule |

### Job Executions

| Method | Path | Description |
|---|---|---|
| GET | `/jobs/:guid/history` | List execution history for a job |
| GET | `/jobs/:guid/schedules/:schedule_guid/history` | List execution history for a job schedule |

### Calls

| Method | Path | Description |
|---|---|---|
| POST | `/calls?app_guid=` | Create a call |
| GET | `/calls?space_guid=` | List all calls in a space |
| GET | `/calls/:guid` | Get a call |
| DELETE | `/calls/:guid` | Delete a call |
| POST | `/calls/:guid/execute` | Execute a call immediately |

### Call Schedules

| Method | Path | Description |
|---|---|---|
| POST | `/calls/:guid/schedules` | Create a schedule for a call |
| GET | `/calls/:guid/schedules` | List schedules for a call |
| DELETE | `/calls/:guid/schedules/:schedule_guid` | Delete a call schedule |

### Call Executions

| Method | Path | Description |
|---|---|---|
| GET | `/calls/:guid/history` | List execution history for a call |
| GET | `/calls/:guid/schedules/:schedule_guid/history` | List execution history for a call schedule |

### Other

| Method | Path | Description |
|---|---|---|
| GET | `/` | Health check |
| GET | `/scheduler-time-zones` | List available timezones |

## CI/CD Pipeline

The project uses [Concourse CI](https://concourse-ci.org/) for automated builds, testing,
and releases. All pipeline configuration lives in the `ci/` directory.

### Pipeline Groups

| Group | Jobs | Purpose |
|---|---|---|
| **build-ocf-scheduler-releases** | build, ship-prerelease, test, prepare, ship-release | Main release pipeline |
| **test-ocf-scheduler-pull-requests** | test-pr | Automated PR testing |
| **version** | major, minor, patch | Manual version bumps |

### Release Pipeline Flow

```
git push to develop
        │
        ▼
      build ──────► ship-prerelease (GitHub pre-release)
        │
        ▼
      test
        │
        ▼
     prepare ────► generates release notes
        │
        ▼
    ship-release ─► merge to main, GitHub release, bump patch version
```

### Pipeline Jobs

| Job | Trigger | Description |
|---|---|---|
| **build** | Automatic on push to `develop` | Cross-compiles binaries, uploads artifacts to S3, bumps RC version |
| **test** | Automatic after build passes | Validates build artifacts |
| **ship-prerelease** | After build passes | Creates a GitHub prerelease with build artifacts |
| **prepare** | Automatic after test passes | Generates release notes from commit messages, pushes to release-notes repo |
| **ship-release** | Manual | Builds final release, merges `develop` into `main`, creates GitHub release, bumps patch version |
| **test-pr** | Automatic on pull request | Runs tests and reports pass/fail status back to GitHub |
| **major** | Manual | Bumps the major version number |
| **minor** | Manual | Bumps the minor version number |
| **patch** | Manual | Bumps the patch version number |

### Pipeline Configuration

Pipeline settings are defined in `ci/settings.yml` and merged with the pipeline templates
in `ci/pipeline/` using [spruce](https://github.com/geofffranks/spruce). Key settings:

- **Source branch**: `develop` (`meta.github.branch`)
- **Release branch**: `main` (`meta.github.main-branch`)
- **Notifications**: Slack webhooks on job failures
- **Artifact storage**: S3 bucket for build artifacts
- **Distribution**: GitHub releases for final and pre-release builds

### How CI Builds Use the Makefile

The CI build script (`ci/scripts/build`) invokes the same Makefile targets used for local
development:

```sh
make release VERSION=$VERSION APP_NAME=$APP_NAME CGO_ENABLED=$CGO_ENABLED
```

The `VERSION` is injected from the Concourse version resource, ensuring consistent
versioning between CI and the released artifacts.

## Contributing

Issues and pull requests are welcome at [github.com/cloudfoundry-community/ocf-scheduler](https://github.com/cloudfoundry-community/ocf-scheduler).

### Branch Strategy

- **`develop`** — Active development branch. All pull requests should target this branch.
- **`main`** — Stable release branch. Merged from `develop` automatically during the
  `ship-release` pipeline job. Do not push directly to `main`.

### Pull Request Workflow

1. Fork the repository and create a feature branch from `develop`
2. Make your changes and ensure tests pass (`make test`)
3. Open a pull request targeting `develop`
4. The Concourse `test-pr` job will automatically run tests and report status on your PR
5. Once approved and merged, the CI pipeline handles building, testing, and releasing

## License

This project is licensed under the [MIT License](LICENSE.md).
