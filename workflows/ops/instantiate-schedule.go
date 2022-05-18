package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func InstantiateSchedule(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	executable := input.Executable

	schedule := &core.Schedule{}

	if err := input.Context.Bind(&schedule); err != nil {
		return dry.Failure("could-not-bind-input")
	}

	schedule.RefGUID = executable.RefGUID()
	schedule.RefType = executable.RefType()

	input.Schedule = schedule

	return dry.Success(input)
}
