package core

import (
	"github.com/ess/dry"
	"github.com/labstack/echo/v4"
)

type Input struct {
	Context    echo.Context
	Services   *Services
	Executable Executable
	Executions []*Execution
	Schedules  []*Schedule
}

func NewInput(context echo.Context, services *Services) *Input {
	return &Input{
		Context:    context,
		Services:   services,
		Executions: make([]*Execution, 0),
		Schedules:  make([]*Schedule, 0),
	}
}

func Inputify(input dry.Value) *Input {
	d := input.(*Input)

	return d
}
