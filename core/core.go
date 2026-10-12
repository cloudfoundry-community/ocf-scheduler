package core

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrNotScheduled means a schedule has no cron entry to remove.
var ErrNotScheduled = errors.New("no such runner")

const TimezoneJsonBase string = "scheduler-timezones.json"
const TimezoneJsonDir string = "/var/vcap/store/scheduler/"

// CronService is an interface to which all Cron providers must conform.
type Timezone struct {
	Name             string   `json:"Name,omitempty"` // omitempty is in case we do a map instead of a slice
	IsServerTimeZone string   `json:"IsServerTimeZone,omitempty"`
	HasDst           bool     `json:"HasDst"`
	Std              string   `json:"Std"`
	Dst              string   `json:"Dst,omitempty"`
	Aliases          []string `json:"Aliases,omitempty"`
	Rules            string   `json:"Rules,omitempty"`
}

type TimezoneSlice []Timezone

type CronService interface {
	Start()
	Stop() context.Context
	Add(Runnable) error
	Delete(Runnable) error
	Count() int
	Analyze(expression, key string, next, prev int, from time.Time) ScheduleAnalysis
	GetTimezones() (*TimezoneSlice, error)
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

	// Instances and Bindings are set when the service broker is enabled.
	Instances InstanceService
	Bindings  BindingService

	// BindingLock orders binding changes against the routes that
	// require_binding guards. The service broker holds it for writing while
	// it changes instances, bindings, and paused schedules, and a guarded
	// route holds it for reading from its binding check until it has stored
	// its work, so an unbind cannot slip between the two.
	BindingLock sync.RWMutex

	// RequireBinding makes the routes that create or run work refuse an app
	// that has no service binding. It needs Bindings.
	RequireBinding bool
}

// InfoService is an interface to allow the grabbing of ad-hoc information from
// an external source of truth.
type InfoService interface {
	GetSpaceGUIDForApp(string) (string, error)

	// ServiceInstanceExists asks Cloud Controller whether a service instance
	// GUID is still known. It answers false only when Cloud Controller says
	// the instance is not found, and an error when it cannot tell, including
	// when ctx ends first.
	ServiceInstanceExists(ctx context.Context, guid string) (bool, error)
}
