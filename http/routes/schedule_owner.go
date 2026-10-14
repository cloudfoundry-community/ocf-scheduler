package routes

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

// A schedule create and a job or call delete do not share a lock, and the
// scheduler can run more than one instance against the same database, so an
// in-process lock would not cover them anyway. Each side instead checks the
// other through the database after its own write. A delete removes the job or
// call and then sweeps its schedules again. A create stores the schedule,
// adds it to cron, and then looks the job or call up again. Whichever of the
// two reads comes second sees the other side's write, so either the sweep
// finds the new schedule or the create finds its owner gone, and the
// schedule never outlives its job or call.

// sweepSchedules deletes the schedules still stored for an executable that
// has just been deleted, which can only be schedules created while the delete
// ran. It tries every schedule it finds and returns how many it could not
// delete from the database, with the errors it met. When the schedules cannot
// be listed at all, it returns zero and the error, because it cannot tell
// whether any were left.
func sweepSchedules(services *core.Services, tag string, executable core.Executable) (int, error) {
	schedules, err := schedulesOf(services, executable)
	if err != nil {
		return 0, fmt.Errorf("could not list schedules left on %s %s: %w", executable.Type(), ownerOf(executable).GUID, err)
	}

	left := 0
	var errs []error
	for _, schedule := range schedules {
		services.Logger.Warn(tag, fmt.Sprintf("deleting schedule %s, created while %s %s was being deleted", schedule.GUID, executable.Type(), ownerOf(executable).GUID))

		if err := dropOrphanSchedule(services, schedule, executable); err != nil {
			left++
			errs = append(errs, fmt.Errorf("could not delete schedule %s left on %s %s: %w", schedule.GUID, executable.Type(), ownerOf(executable).GUID, err))
		}
	}

	return left, errors.Join(errs...)
}

// confirmOwner runs after a create has stored a schedule and added it to
// cron. When the schedule's job or call is gone, it takes the schedule back
// out and returns the status to answer with. Otherwise it returns zero.
//
// A lookup that fails for another reason keeps the schedule. The job or
// call was there when the create began, and answering with an error would
// invite a retry that stores a second schedule beside this one. If the
// owner has gone after all, the orphan schedules list shows the schedule.
func confirmOwner(services *core.Services, tag string, schedule *core.Schedule, executable core.Executable) (int, any) {
	_, err := findOwner(services, executable)
	if err == nil {
		return 0, nil
	}

	if !errors.Is(err, core.ErrNotFound) {
		services.Logger.Warn(tag, fmt.Sprintf("could not confirm %s %s still exists after creating schedule %s, so the schedule is kept: %v", executable.Type(), ownerOf(executable).GUID, schedule.GUID, err))
		return 0, nil
	}

	services.Logger.Warn(tag, fmt.Sprintf("%s %s was deleted while schedule %s was created, so the schedule is removed", executable.Type(), ownerOf(executable).GUID, schedule.GUID))
	if err := dropOrphanSchedule(services, schedule, executable); err != nil {
		services.Logger.Error(tag, fmt.Sprintf("could not remove schedule %s, whose %s %s is gone: %v", schedule.GUID, executable.Type(), ownerOf(executable).GUID, err))
	}

	return http.StatusNotFound, ""
}

// dropOrphanSchedule deletes a schedule whose job or call is known to be
// gone. DeletingASchedule keeps the cron entry when the row delete fails,
// which is right while the owner exists, but here the entry would go on
// running a job or call that no longer exists. So the entry comes out
// whatever happens to the row. Startup and resume load a schedule only
// through its job or call, so a row left behind never returns to cron, and
// the orphan schedules list shows it.
func dropOrphanSchedule(services *core.Services, schedule *core.Schedule, executable core.Executable) error {
	err := workflows.DeletingASchedule(services, schedule, executable)
	if err == nil {
		return nil
	}

	if cronErr := services.Cron.Delete(runOf(services, schedule, executable)); cronErr != nil && !errors.Is(cronErr, core.ErrNotScheduled) {
		return errors.Join(err, cronErr)
	}

	return err
}

// runOf builds the run that cron holds for schedule.
func runOf(services *core.Services, schedule *core.Schedule, executable core.Executable) core.Runnable {
	if job, err := executable.ToJob(); err == nil {
		return core.NewJobRun(job, schedule, services)
	}

	call, _ := executable.ToCall()
	return core.NewCallRun(call, schedule, services)
}

// findOwner looks up the stored job or call behind executable again.
func findOwner(services *core.Services, executable core.Executable) (core.Executable, error) {
	if job, err := executable.ToJob(); err == nil {
		return services.Jobs.Get(job.GUID)
	}

	call, _ := executable.ToCall()
	return services.Calls.Get(call.GUID)
}
