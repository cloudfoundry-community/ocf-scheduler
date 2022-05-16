package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func DeleteSchedule(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	executable := input.Executable
	tag := "ops.delete-schedule"
	var run core.Runnable

	if len(input.Schedules) != 1 {
		input.Services.Logger.Error(tag, "expected exactly one schedule")
		return dry.Failure("schedule-count-mismatch")
	}

	schedule := input.Schedules[0]

	switch executable.RefType() {
	case "call":
		call, err := executable.ToCall()
		if err != nil {
			input.Services.Logger.Error(tag, "got a call that isn't a call")
			return dry.Failure("executable-type-mistmatch")
		}

		run = core.NewCallRun(call, schedule, input.Services)
	default:
		job, err := executable.ToJob()
		if err != nil {
			input.Services.Logger.Error(tag, "got a job that isn't a job")
			return dry.Failure("executable-type-mismatch")
		}

		run = core.NewJobRun(job, schedule, input.Services)
	}

	if input.Services.Cron.Delete(run) != nil {
		input.Services.Logger.Error(
			tag,
			fmt.Sprintf(
				"could not unschedule the run for %s",
				schedule.GUID,
			),
		)

		return dry.Failure("unschedule-failure")
	}

	if input.Services.Schedules.Delete(schedule) != nil {
		input.Services.Logger.Error(
			tag,
			fmt.Sprintf(
				"could not delete schedule with GUID %s",
				schedule.GUID,
			),
		)

		return dry.Failure("delete-schedule-failed")
	}

	return dry.Success(input)
}
