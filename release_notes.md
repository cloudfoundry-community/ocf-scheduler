# Breaking Changes

* **The Day of Month (DOM) and Day of Week (DOW) fields are now AND'd together.** When both
  fields are restricted, a job fires only when *both* match — consistent with how every
  other cron field behaves. Previously both fields were OR'd when neither was `*`.

  Existing schedules that restrict both fields will fire **less often** than they did in
  v2.0.0. For example, `0 0 13 * FRI` previously fired on the 13th of every month *and* on
  every Friday; it now fires only on a Friday the 13th. Review any schedule that sets both
  DOM and DOW to a value other than `*`.

  The new semantics make previously impossible patterns expressible:

  | expression | meaning |
  |:-----------|:--------|
  | `0 0 25-31 * FRI` | last Friday of the month |
  | `0 0 1-7 * MON`   | first Monday of the month |

  Behaviour when either field is `*` is unchanged — the restricted field governs.
* `WORKER_NUM` has been renamed to `SCHEDULER_WORKERS`. Deployments setting the old
  variable will silently fall back to the default worker count.
* The cron engine changed from `robfig/cron` v3.0.1 to `netresearch/go-cron`.
* Release assets are now per-platform tarballs rather than a single Linux binary.
  v2.0.0 shipped `ocf-scheduler-2.0.0-linux-amd64`; v2.0.1 ships four archives, each
  containing `scheduler`, `tzlist`, and their `.sha1`/`.sha256` files.

# Bug Fixes

* Fixed `Scan` failures caused by `SELECT *` column ordering. Adding the log rate field
  changed column order and broke row scanning; all queries now use named columns so new
  fields cannot reorder existing ones.
* Corrected the database migration filename.
* Renamed `postgres/scheudle_service.go` to `schedule_service.go`.
* Cleared `go vet` diagnostics across the tree.
* Fixed the `release-workdir` `-o` flag to target a directory.

# Improvements

## CAPI v3 Migration

* Upgraded from the community `go-cfclient` v2 to the official `cloudfoundry/go-cfclient/v3`
  (`v3.0.0-alpha.20`).
* Removed the legacy `/v2/info` endpoint from the mock CF API server.

## Cyclic Ranges

The OCF Scheduler ranges now support cyclic ranges for the minutes, hours, days of the week, and months fields.
A cyclic range means the start value can be larger than the ending value.   This allows cron expressions
to be more expressive instead of creating multiple ranges.  It also makes fields with named values work without
needing to know the actual values.

Examples
|expression|field|description|
|:---------|:-------: |-----------|
|57-3      | minutes | 57, 58, 59, 0, 1, 2, 3|
|23-1      |  hours  | 23, 0, 1 (spans midnight)|
|FRI-MON   |   DOW   | FRI, SAT, SUN, MON (spans the weekend)|
|DEC-FEB   |  month  | DEC, JAN, FEB (spans year boundary)|
|26-3      |  DOM    | Wraps correctly for every month

### Cyclic Step Expressions

Cyclic ranges also work in step expressions

|expression|field|description|
|:---------|:-------: |-----------|
|20-2/2    |hours     | every 2 hours from 8 pm(20) to 2 am|

## Scheduling and Jobs

* Added a **log rate** parameter.
* Reworked scheduler worker handling; the worker count is now set by `SCHEDULER_WORKERS`.
* Added a timezone listing route.

## Logging and Diagnostics

* `LOG_LEVEL` now accepts both `warn` and `warning`, and its processing is explicit and
  case-insensitive.
* Database migrations are now written to the log file, with a clearer migration message.
* Job name is included in application log messages.
* Switched to `log/slog`, replaced `ioutil` with `os`, and moved to `filepath` builders.

## Testing

* Added unit tests across the presenter, core, logger, and cf packages.
* Raised `tzposix` coverage from 79.5% to 96.9%.
* Added `cmd/scheduler` coverage for `ErrorString`, `createBuildMeta`, and `createSemVer`.

## Build and Release

* Builds now produce artifacts for linux/amd64, linux/arm64, darwin/amd64, and darwin/arm64.
* Release binaries are produced by re-linking the objects compiled during the build rather
  than recompiling, so the released binaries are provably the same code as the tested
  release candidate — only the version stamp differs.
* SHA-1 and SHA-256 checksums are published alongside every binary.

# Software Components

## Core Components

| Release | Version | Release Date | Type | Changed |
| ------- | ------- | ------------ | ---- | :-----: |
| Go | 1.26.0 | | toolchain | [X](## was 1.24) |
| go-cron | [v0.14.0][go-cron]| |source|[X](## was robfig/cron v3.0.1)
| go-cfclient | [v3.0.0-alpha.20][go-cfclient]| |source|[X](## was community go-cfclient v2)

[go-cron]: https://github.com/netresearch/go-cron/releases/tag/v0.14.0
[go-cfclient]: https://github.com/cloudfoundry/go-cfclient/releases/tag/v3.0.0-alpha.20
