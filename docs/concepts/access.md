# Who can see and change what

The scheduler uses Cloud Foundry's own users and roles. Every request
carries the caller's UAA token, and the scheduler asks Cloud Foundry about
the caller's roles on every request, so a role change takes effect at once.
When the scheduler can't look the roles up, it refuses the request.

## Space developers

A job or a call belongs to the space of the app it was created for. A
caller can see, create, run, and delete jobs, calls, and schedules only in
a space where they hold the Space Developer role. That holds for reading
too, so the history of a job is visible only to the developers of its
space.

Space managers and space auditors can't see or change anything in the
scheduler. To anyone who isn't a developer in its space, a job or a call
looks as though it doesn't exist, so the scheduler never reveals what
another space has scheduled.

A few things hold no space's data: listing the time zones the scheduler
accepts, and checking a cron expression without naming a job or a call.
They're open to any caller who is a Space Developer or a Space Manager in
at least one space.

## Admins

A caller whose token carries the `cloud_controller.admin` scope can act in
every space. Only an admin can list and remove orphans, the jobs and calls
whose app is gone and the schedules whose job or call is gone, and only
with the token of an admin user, such as the one `cf oauth-token` prints. A
token that a UAA client gets for itself is refused, even when it carries
the admin scope.

## A call's authorization header

Every call stores an `Authorization` header to send to its endpoint. The
scheduler sends it on every run and never shows it again, not to the
call's developers and not to an admin.

## The service broker's part

The operators who deploy the scheduler can also run it as a Cloud Foundry
service. When they do, the roles above still decide who can do what, and
the service adds the following.

- **Service instance.** Creating the scheduler service in a space turns
  the scheduler on for that space. A space holds one instance at most.
- **Binding.** Binding an app to the instance lets that app be scheduled.
  The binding holds the scheduler's address and no secret, because each
  caller still signs in with their own token.
- **Unbinding.** When an app loses its last binding, its schedules are
  paused, and binding it again resumes them. See *Schedules, cron
  expressions, and time zones → Enabled and disabled schedules*.
- **Deleting the instance.** This unbinds every app and pauses their
  schedules. The space's jobs, calls, schedules, and history all stay.

Operators can also require a binding. Then an app that isn't bound can't
have jobs or calls created for it, schedules added to them, or runs
started by hand. Seeing, listing, and deleting stay open, so a developer
can still clean up what an unbound app has. An admin is never held to the
binding.
