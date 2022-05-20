# scheduler-for-ocf #

This project is a clean room, drop-in replacement for the scheduler-for-pcf service API.

We've aimed to make it behave as close to the documented behavior of the original as possible, but there is a fairly good chance that there are a few unintended differences.

## Glossary ##

* **Call**: An Executable that describes a URL to which one wishes to make a request
* **Executable**: An object that can be executed by the scheduler
* **Execution**: An object that describes the actual execution of an Executable
* **Job**: An Executable that describes a command one wishes to run in an existing app's context
* **Schedule**: An object that describes the period over which an Executable should be executed

## REST(ish) API ##

The scheduler service exposes an HTTP REST API for client interaction.

### Authorization ###

With the exception of the root endpoint (which responds to `GET` requests as a health check), all endpoints require that a valid bearer token be provided in the `Authorization` header for the request in the form `Bearer tokentokentokentokentoken` (just as it appears in ~/.cf/config.json).

The cf user associated with the provided token must be a space developer or space manager in order to use the API.

### Errors ###

It's worth noting up front that almost every endpoint in the API presents at least one error scenario. In order to preserve the behavior of the system we are mimicing, when an error is returned from an endpoint, the only visibiilty into what went wrong to the client is the HTTP status code returned. The response body for an error is empty.

### Jobs ###

A Job is an executable object that describes a command that one wishes to run within an app's context. Its JSON representation looks like this:

```json
{
  "guid" : "11111111-1111-1111-1111-111111111111",
  "name" : "command-name",
  "command" : "echo 'I am a command'",
  "disk_in_mb" : 1024,
  "memory_in_mb" : 1024,
  "state" : "PENDING",
  "created_at" : "2022-05-20T21:21:44.763885632Z",
  "updated_at" : "2022-05-20T21:21:44.763885632Z",
  "app_guid" : "22222222-2222-2222-2222-222222222222",
  "space_guid" : "33333333-3333-3333-3333-333333333333"

}
```

A note on the "state" field: the initial state for a job is `PENDING`. Over the course of its lifetime, a job's state can also be either `SUCCEEDED` or `FAILED`, based on its most recent execution state.

#### Creating a Job ####

#### Deleting a Job ####

#### Executing a Job ####

#### Getting a Job ####

#### Getting All Jobs ####

#### Getting a Job's Execution History ####

#### Getting a Job's Scheduled Execution History ####

#### Scheduling a Job ####

#### Unscheduling a Job ####

### Calls ###

#### Creating a Call ####

#### Deleting a Call ####

#### Executing a Call ####

#### Getting a Call ####

#### Getting All Calls ####

#### Getting a Call's Execution History ####

#### Getting a Call's Scheduled Execution History ####

#### Scheduling a Call ####

#### Unscheduling a Call ####


## History ##

* v0.0.9 - POST calls don't GET none
* v0.0.8 - all-jobs logs
* v0.0.7 - Now with root as health
* v0.0.6 - Corrected Call field order
* v0.0.5 - Now with listen port configuration
* v0.0.4 - Now with working task creation
* v0.0.3 - Better console errors
* v0.0.1 - initial dev release
