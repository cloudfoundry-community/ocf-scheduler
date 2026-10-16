# Executions and their states

An execution is the record of one run of a job or a call. The scheduler
creates one each time a schedule fires and each time someone runs a job or
a call by hand, and it records how the run ended.

## What an execution records

- **State.** `PENDING`, `SUCCEEDED`, or `FAILED`, as *Execution states*
  describes.
- **Schedule.** The schedule that started the run. A run started by hand
  has none.
- **Scheduled time.** The minute the schedule fired, in UTC. A run started
  by hand has none.
- **Start and end times.** When the scheduler began the run and when it
  recorded the outcome.
- **Task.** For a job, the GUID of the Cloud Foundry task that the run
  created, which ties the execution to the task's logs in Cloud Foundry.
- **Message.** A short note on the outcome, such as the reason Cloud
  Foundry gave for a failed task, or why the scheduler failed the run.

## Execution states

| State | Meaning |
|---|---|
| `PENDING` | The run has been recorded and hasn't ended yet. That includes a run whose task or call is under way |
| `SUCCEEDED` | The task or the call succeeded |
| `FAILED` | The task or the call failed, or the scheduler couldn't carry out the run. The message says which |

An execution has no running state of its own. It stays `PENDING` from the
moment it's recorded until the scheduler records the outcome, which for a
long task can be hours. `SUCCEEDED` and `FAILED` are final.

## How a job's outcome is decided

A job runs as a Cloud Foundry task on the job's app. The scheduler creates
the task, records its GUID on the execution, and then asks Cloud Foundry
for the task's state every five seconds until the task ends.

- The execution succeeds only when the task ends `SUCCEEDED`.
- When the task ends `FAILED`, the execution fails, with the task's failure
  reason as its message.
- When Cloud Foundry refuses to create the task, or stops answering about
  it, the execution fails.
- When the task is still waiting to start after 15 minutes, the scheduler
  asks Cloud Foundry to cancel it and fails the execution with a message
  saying the task never started. A task that has started has no time
  limit.

## How a call's outcome is decided

A call is one HTTP `POST` to the call's URL. The execution succeeds when
the endpoint answers with a `2xx` status, and it fails on any other status,
on a connection or TLS error, or when no answer arrives within 20 seconds.
The scheduler doesn't retry a failed call. The next run is the next time
its schedule fires.

## Runs that the scheduler skips

The scheduler would rather miss a run than start a task or a call that
nothing records. When it can't store a new execution, it skips the run and
there's no execution for it, only an entry in the scheduler's log. When it
stores the execution but can't mark it started, it skips the run and fails
the execution with a message saying so. If the scheduler's database stays
unreachable throughout, that execution can stay `PENDING` for good, and
then it stands for a run that never happened. See *Operator guide →
Executions that stay PENDING*.

## Execution history

The history of a job or a call lists all of its executions, from all of
its schedules and its runs by hand. The history of one schedule lists only
the runs that schedule started.

Execution history outlives the app. When an app is deleted, its jobs,
calls, schedules, and executions stay in the scheduler until an operator
removes them. See *Operator guide → Finding and removing orphaned jobs,
calls, and schedules*.

## How long executions are kept

**TBD.** Once a day the scheduler deletes executions that ended
`SUCCEEDED` or `FAILED` longer ago than a retention period. The code sets
that period to 150 days, while the comment beside it says 30 days, so the
intended period is to be confirmed. The period can't be configured.
Executions still `PENDING` are never deleted.
