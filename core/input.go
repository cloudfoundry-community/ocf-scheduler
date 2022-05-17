package core

import (
	"github.com/ess/dry"
	"github.com/labstack/echo/v4"
)

type Input struct {
	// Helpers
	Context  echo.Context
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

func NewInput(context echo.Context, services *Services) *Input {
	return &Input{
		Context:             context,
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
