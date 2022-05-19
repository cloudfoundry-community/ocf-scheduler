package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func PersistSchedule(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	services := input.Services

	schedule, err := services.Schedules.Persist(input.Schedule)
	if err != nil {
		services.Logger.Error(
			"ops.persist-schedule",
			fmt.Sprintf("could not persist the schedule: %s", err.Error()),
		)

		return dry.Failure(failures.PersistScheduleFailure)
	}

	input.Schedule = schedule

	return dry.Success(input)
}
