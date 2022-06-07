package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func LoadSchedule(raw dry.Value) dry.Result {
	tag := "ops.load-schedule"
	input := core.Inputify(raw)
	guid := input.Data["scheduleGUID"]

	if guid == "" {
		input.Services.Logger.Error(
			tag,
			"no schedule guid provided",
		)
		return dry.Failure(failures.NoScheduleGUID)
	}

	schedule, err := input.Services.Schedules.Get(guid)
	if err != nil {
		input.Services.Logger.Error(
			tag,
			fmt.Sprintf("could not find schedule with guid %s", guid),
		)

		return dry.Failure(failures.NoSuchSchedule)
	}

	input.Schedule = schedule

	return dry.Success(input)
}
