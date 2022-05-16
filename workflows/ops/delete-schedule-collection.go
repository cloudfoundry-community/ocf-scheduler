package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func DeleteScheduleCollection(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	for _, schedule := range input.ScheduleCollection {
		// hey, let's call another op from this op
		secondary := core.NewInput(input.Context, input.Services)
		secondary.Executable = input.Executable
		secondary.Schedule = schedule

		result := DeleteSchedule(secondary)

		if result.Failure() {
			return dry.Failure(result.Error())
		}
	}

	return dry.Success(input)
}
