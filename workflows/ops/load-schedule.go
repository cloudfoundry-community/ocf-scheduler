package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func LoadSchedule(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	guid := input.Data["scheduleGUID"]

	schedule, err := input.Services.Schedules.Get(guid)
	if err != nil {
		input.Services.Logger.Error(
			"ops.load-schedule",
			fmt.Sprintf("could not find schedule with guid %s", guid),
		)

		return dry.Failure("no-such-schedule")
	}

	input.Schedule = schedule

	return dry.Success(input)
}
