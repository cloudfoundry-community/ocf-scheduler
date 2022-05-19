package core

import (
	"github.com/ess/dry"
)

type Input struct {
	// Helpers
	Services *Services

	// Single records
	Executable Executable
	Execution  *Execution
	Schedule   *Schedule

	// Collections
	JobCollection       []*Job
	CallCollection      []*Call
	ExecutionCollection []*Execution
	ScheduleCollection  []*Schedule

	// Misc Data
	Data map[string]string
}

func NewInput(services *Services) *Input {
	return &Input{
		Services:            services,
		JobCollection:       make([]*Job, 0),
		CallCollection:      make([]*Call, 0),
		ExecutionCollection: make([]*Execution, 0),
		ScheduleCollection:  make([]*Schedule, 0),
		Data:                make(map[string]string),
	}
}

func Inputify(input dry.Value) *Input {
	d := input.(*Input)

	return d
}

func (input *Input) WithAuth(auth string) *Input {
	input.Data["auth"] = auth

	return input
}

func (input *Input) WithGUID(guid string) *Input {
	input.Data["guid"] = guid

	return input
}

func (input *Input) WithExecutable(exe Executable) *Input {
	input.Executable = exe

	return input
}

func (input *Input) WithSchedule(sch *Schedule) *Input {
	input.Schedule = sch

	return input
}

func (input *Input) WithExecution(exe *Execution) *Input {
	input.Execution = exe

	return input
}

func (input *Input) WithScheduleGUID(guid string) *Input {
	input.Data["scheduleGUID"] = guid

	return input
}

func (input *Input) WithAppGUID(guid string) *Input {
	input.Data["appGUID"] = guid

	return input
}

func (input *Input) WithSpaceGUID(guid string) *Input {
	input.Data["spaceGUID"] = guid

	return input
}
