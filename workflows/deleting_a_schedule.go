package workflows

import (
	"fmt"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func DeletingASchedule(services *core.Services, schedule *core.Schedule, executable core.Executable) error {
	var run core.Runnable
	tag := "deleting-a-schedule"

	switch executable.Type() {
	case "call":
		call, err := executable.ToCall()
		if err != nil {
			doh := "got a call that isn't a call"
			services.Logger.Error(tag, doh)
			return fmt.Errorf(doh)
		}

		run = core.NewCallRun(call, schedule, services)
	default:
		job, err := executable.ToJob()
		if err != nil {
			doh := "got a job that isn't a job"
			services.Logger.Error(tag, doh)
			return fmt.Errorf(doh)
		}

		run = core.NewJobRun(job, schedule, services)
	}

	if services.Cron.Delete(run) != nil {
		doh := fmt.Sprintf("could not unschedule the run for %s", schedule.GUID)
		services.Logger.Error(tag, doh)
		return fmt.Errorf(doh)
	}

	if services.Schedules.Delete(schedule) != nil {
		doh := fmt.Sprintf("could not delete schedule with GUID %s", schedule.GUID)
		services.Logger.Error(tag, doh)
		return fmt.Errorf(doh)
	}

	return nil
}
