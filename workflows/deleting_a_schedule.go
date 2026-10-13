package workflows

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
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
			return fmt.Errorf("%s", doh)
		}

		run = core.NewCallRun(call, schedule, services)
	default:
		job, err := executable.ToJob()
		if err != nil {
			doh := "got a job that isn't a job"
			services.Logger.Error(tag, doh)
			return fmt.Errorf("%s", doh)
		}

		run = core.NewJobRun(job, schedule, services)
	}

	// Delete the row before the cron entry. A rebind that lists the
	// schedule before the row goes may put it back in cron, and the
	// Cron.Delete below then takes it out again, or the rebind finds the row
	// gone when it resumes and takes it out itself. A rebind that lists the
	// schedules after the row goes never sees it. The other order lets a
	// rebind re-add the entry between the two steps and leave it firing for
	// a schedule that no longer exists. A row delete or a row lookup that
	// fails returns before the cron entry is touched.
	if services.Schedules.Delete(schedule) != nil {
		doh := fmt.Sprintf("could not delete schedule with GUID %s", schedule.GUID)
		services.Logger.Error(tag, doh)
		return fmt.Errorf("%s", doh)
	}

	// A schedule with no cron entry, such as one that is disabled or that an
	// unbind paused, is already out of cron.
	if err := services.Cron.Delete(run); err != nil && !errors.Is(err, core.ErrNotScheduled) {
		doh := fmt.Sprintf("could not unschedule the run for %s", schedule.GUID)
		services.Logger.Error(tag, doh)
		return fmt.Errorf("%s", doh)
	}

	return nil
}
