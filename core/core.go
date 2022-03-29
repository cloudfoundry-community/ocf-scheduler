package core

import "context"

// CronService is an interface to which all Cron providers must conform.
type CronService interface {
	Start()
	Stop() context.Context
	Add(Runnable) error
	Delete(Runnable) error
	Count() int
}

// Executable is an intermediary interface to allow all things that can be
// executed to be processed in roughly the same way at the high level.
type Executable interface {
	Type() string
	ToCall() (*Call, error)
	ToJob() (*Job, error)
}

// Services is just a big collection of all of the services one may need for
// any given workflow. It's the obligatory god object for this project.
type Services struct {
	Info       InfoService
	Jobs       JobService
	Calls      CallService
	Schedules  ScheduleService
	Workers    WorkerService
	Runner     RunService
	Executions ExecutionService
	Cron       CronService
	Logger     LogService
	Auth       AuthService
}

// InfoService is an interface to allow the grabbing of ad-hoc information from
// an external source of truth.
type InfoService interface {
	GetSpaceGUIDForApp(string) (string, error)
}
