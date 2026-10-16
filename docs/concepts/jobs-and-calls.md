# Jobs and calls

The scheduler runs two kinds of work. A job runs a command in an app, and
a call sends an HTTP request to a URL. Both belong to an app, both run on
schedules or by hand, and both keep a history of their runs.

## Jobs

A job is a command that the scheduler runs as a Cloud Foundry task on the
job's app. The task runs with the app's droplet and environment, the same
as a task started with `cf run-task`.

A job has:

- **A name**, unique among the app's jobs.
- **A command**, run in the app's container.
- **Disk and memory limits** for the task, each 1024 MB unless the job
  sets its own.
- **A log rate limit** for the task, in bytes per second. The cf CLI
  plugin sets `1024M` unless told otherwise. **TBD:** a job created
  without one asks Cloud Foundry for a limit of `0`, and what Cloud
  Foundry does with that is to be confirmed.
- **A state**, as *Job state* describes.

## Job state

**TBD.** A job carries a state that the scheduler sets to `PENDING` when
the job is created. Today nothing changes it after that, so every job
reads `PENDING`, however often it has run and however its runs ended. The
outcome of each run is on its execution, as *Executions and their states*
describes. What a job's state should hold is to be decided.

## Calls

A call is an HTTP `POST` that the scheduler sends to the call's URL. It
lets a schedule reach any endpoint, such as a route of the call's own app,
another service, or something outside Cloud Foundry.

A call has:

- **A name**, unique among the app's calls.
- **A URL**, the endpoint the request goes to.
- **An authorization header**, sent as the request's `Authorization`
  header, such as `bearer <token>`. Every call needs one.

## What a call's endpoint receives

Each run sends one request, the same every time:

- The method is `POST`, and the request has no body.
- The `Authorization` header holds the call's authorization header,
  exactly as it was stored.
- The scheduler checks the endpoint's TLS certificate, and an endpoint
  whose certificate it can't verify fails the run.
- The endpoint has 20 seconds to answer. A `2xx` status counts as
  success, and anything else, or no answer in time, as failure.

The scheduler doesn't retry a failed request, so an endpoint that misses
one run gets the next one when the schedule fires again. A request can
also arrive again when someone runs the call by hand.

The scheduler stores the authorization header and sends it on every run,
but it never shows it again, to anyone. To change it, create a new call.

## How jobs and calls belong to apps

Every job and call is created for one app, and it belongs to that app's
space for as long as it exists. Neither can be changed after it's created;
to change one, delete it and create it again.

Deleting a job or a call deletes its schedules, so it stops running. It
doesn't delete its execution history, which the scheduler removes later,
as *Executions and their states → How long executions are kept*
describes.

Deleting the app doesn't delete its jobs and calls. A job whose app is
gone fails each time it runs, and a call whose app is gone goes on sending
its request, because its URL doesn't depend on the app. Both stay until an
operator removes them, as *Operator guide → Finding and removing orphaned
jobs, calls, and schedules* describes.
