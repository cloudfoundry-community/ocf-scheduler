# Schedules, cron expressions, and time zones

A schedule says when a job or a call runs. This page covers what a schedule
is and how the scheduler reads its cron expression. For the expression
syntax itself, field by field, see the
[cron expression reference](https://github.com/cloudfoundry-community/ocf-scheduler-cf-plugin/blob/develop/cron-expression-reference.md)
in the cf CLI plugin.

## What a schedule is

A schedule is one cron expression attached to one job or one call. A job
or a call can have any number of schedules, and each one fires on its own.
A run that a schedule starts records the schedule on its execution, so the
history of a single schedule can be read apart from the rest.

A schedule can't be changed after it's created. To change when something
runs, add a schedule with the new expression and delete the old one.
Deleting a job or a call deletes all of its schedules.

## Enabled and disabled schedules

A schedule is either enabled or disabled, and only an enabled schedule
fires. The cf CLI plugin creates every schedule enabled.

The scheduler disables schedules itself in one case. When the service
broker is in use and an app loses its last binding, the scheduler disables
every enabled schedule on that app's jobs and calls and marks each one as
paused by the unbind. Binding the app again enables only the schedules
that carry that mark. A schedule that was already disabled before the
unbind stays disabled. See *Who can see and change what → The service
broker's part*.

## How the scheduler reads an expression

The scheduler checks every expression when a schedule is created, and it
refuses one that doesn't parse, that can never fire, or that fires more
often than the minimum interval allows. The same check can be run without
creating anything, which reports the errors and warnings, a plain
description of the expression, and its next and previous run times.

- **Fields.** An expression has five, six, or seven fields. Five fields
  are minute, hour, day of month, month, and day of week. Six fields add
  seconds at the front, unless the last field is 100 or more, in which
  case it's a year. Seven fields are seconds through year.
- **Day of month and day of week.** When both are restricted, a run needs
  both to match. `0 9 13 * FRI` runs only on a Friday the 13th.
- **`H`.** An `H` in a field stands for a value that the scheduler picks
  from the job's or call's GUID. The same job always gets the same value,
  and different jobs get spread out, so that many schedules written as
  `H * * * *` don't all fire in the same minute. A check run without a job
  or call shows an example value only.
- **Expressions that never fire.** An expression such as `0 0 30 2 *`
  (February 30th) parses but never fires. The scheduler looks five years
  ahead, so an expression whose next run is further away than that is
  treated the same way.

## Minimum interval

**TBD.** The scheduler reads a minimum interval from
`SCHEDULER_MIN_INTERVAL`, a Go duration such as `5m`, and refuses a new
schedule that would fire more often than that. It is off when unset or
`0`. A stored schedule that breaks a minimum set later still loads and
runs, with a warning in the log, so raising the minimum never stops
existing schedules. This setting is not yet in the service README's
configuration table.

## Time zones

A schedule runs in the time zone that its expression names with a
`CRON_TZ=` prefix, such as `CRON_TZ=Europe/Berlin 0 9 * * *`. An
expression without the prefix runs in the scheduler server's own time
zone, which the operators who deploy the scheduler choose. The scheduler
lists the time zone names it accepts.

The time zone rules come from the time zone data on the scheduler's
server, so a country's change to its clocks takes effect when the
operators update that data. See *Operator guide → Where time zone data
comes from*.

## Daylight saving time

When the clocks change, a schedule's wall-clock time can be missing or can
happen twice.

- **Clocks go forward by an hour.** A run in the skipped hour moves forward
  by an hour, so a 2:30 run happens at 3:30 that day.
- **Clocks go forward by less than an hour, or at midnight.** Some zones
  skip 30 minutes, such as `Australia/Lord_Howe`, or skip midnight, such
  as `America/Santiago`. A run in that gap doesn't happen that day.
- **Clocks go back.** A run once a day in the repeated hour happens once.
  A schedule that fires several times within the repeated hour, such as
  every 5 minutes, fires in both copies of the hour.

A schedule written in `UTC` never meets any of these.
