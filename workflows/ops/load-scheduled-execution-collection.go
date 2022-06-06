package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func LoadScheduledExecutionCollection(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	schedule := input.Schedule

	if schedule == nil {
		return dry.Failure(failures.ScheduleNotProvided)
	}

	input.ExecutionCollection = input.Services.Executions.BySchedule(
		schedule,
	)

	return dry.Success(input)
}
