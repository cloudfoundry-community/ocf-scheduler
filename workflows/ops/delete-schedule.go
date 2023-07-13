package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
)

func DeleteSchedule(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	executable := input.Executable
	schedule := input.Schedule
	tag := "ops.delete-schedule"
	var run core.Runnable

	switch executable.RefType() {
	case "call":
		call, err := executable.ToCall()
		if err != nil {
			input.Services.Logger.Error(tag, "got a call that isn't a call")
			return dry.Failure(failures.ExecutableTypeMismatch)
		}

		run = core.NewCallRun(call, schedule, input.Services)
	default:
		job, err := executable.ToJob()
		if err != nil {
			input.Services.Logger.Error(tag, "got a job that isn't a job")
			return dry.Failure(failures.ExecutableTypeMismatch)
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

		return dry.Failure(failures.UnscheduleFailure)
	}

	if input.Services.Schedules.Delete(schedule) != nil {
		input.Services.Logger.Error(
			tag,
			fmt.Sprintf(
				"could not delete schedule with GUID %s",
				schedule.GUID,
			),
		)

		return dry.Failure(failures.DeleteScheduleFailed)
	}

	return dry.Success(input)
}
