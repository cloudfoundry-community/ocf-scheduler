package workflows

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// PausingAnApp disables every enabled schedule on the app's jobs and calls,
// marks each with core.PausedByUnbind, and takes it out of cron. Jobs, calls,
// schedules, and history all stay. It returns how many schedules it paused.
//
// It also takes the app's other disabled schedules out of cron, because the
// create routes put a schedule in cron even when it was created disabled, and
// because an earlier attempt may have paused a schedule and then failed. A
// retry therefore finishes what a failed attempt started. It returns an
// error when any disabled schedule may still be in cron, so the caller can
// keep the binding and let Cloud Controller retry.
func PausingAnApp(services *core.Services, appGUID string) (int, error) {
	tag := "pausing-an-app"

	paused, err := services.Schedules.Pause(appGUID, core.PausedByUnbind)
	if err != nil {
		doh := fmt.Sprintf("could not pause the schedules of app %s: %v", appGUID, err)
		services.Logger.Error(tag, doh)
		return 0, fmt.Errorf("%s", doh)
	}

	schedules, err := services.Schedules.ByApp(appGUID)
	if err != nil {
		// Still take out what the pause just changed, so that only the
		// sweep is left for the retry.
		for _, schedule := range paused {
			_ = unschedule(services, schedule)
		}

		doh := fmt.Sprintf("paused %d schedules for app %s but could not list its schedules to take them out of cron: %v", len(paused), appGUID, err)
		services.Logger.Error(tag, doh)
		return 0, fmt.Errorf("%s", doh)
	}

	failed := 0
	for _, schedule := range schedules {
		if schedule.Enabled {
			continue
		}

		if err := unschedule(services, schedule); err != nil {
			failed++
			services.Logger.Error(tag, fmt.Sprintf("could not take schedule %s out of cron: %v", schedule.GUID, err))
		}
	}

	if failed > 0 {
		return 0, fmt.Errorf("could not take %d schedules of app %s out of cron", failed, appGUID)
	}

	services.Logger.Info(tag, fmt.Sprintf("paused %d schedules for app %s", len(paused), appGUID))

	return len(paused), nil
}

// ResumingAnApp enables only the app's schedules that PausingAnApp paused,
// clears their mark, and puts them back in cron. A schedule a user disabled
// stays disabled. It returns how many schedules it resumed.
func ResumingAnApp(services *core.Services, appGUID string) (int, error) {
	tag := "resuming-an-app"

	resumed, err := services.Schedules.Resume(appGUID, core.PausedByUnbind)
	if err != nil {
		doh := fmt.Sprintf("could not resume the schedules of app %s: %v", appGUID, err)
		services.Logger.Error(tag, doh)
		return 0, fmt.Errorf("%s", doh)
	}

	for _, schedule := range resumed {
		run, err := runFor(services, schedule)
		if err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("resumed schedule %s but could not load its %s: %v", schedule.GUID, schedule.RefType, err))
			continue
		}

		// Startup loads the same schedule the same way, so a failure here
		// is logged and left as startup would leave it.
		if err := services.Cron.Add(run); err != nil {
			services.Logger.Error(tag, fmt.Sprintf("could not put schedule %s back in cron: %v", schedule.GUID, err))
		}
	}

	services.Logger.Info(tag, fmt.Sprintf("resumed %d schedules for app %s", len(resumed), appGUID))

	return len(resumed), nil
}

// unschedule takes a schedule out of cron. Cron finds the entry by the
// schedule's GUID alone, so it doesn't load the job or call, and a lost
// database cannot leave the entry behind. A schedule that has no entry is
// already where it should be.
func unschedule(services *core.Services, schedule *core.Schedule) error {
	var run core.Runnable
	if schedule.RefType == "call" {
		run = core.NewCallRun(&core.Call{GUID: schedule.RefGUID}, schedule, services)
	} else {
		run = core.NewJobRun(&core.Job{GUID: schedule.RefGUID}, schedule, services)
	}

	if err := services.Cron.Delete(run); err != nil && !errors.Is(err, core.ErrNotScheduled) {
		return err
	}

	return nil
}

// runFor loads the job or call behind a schedule and wraps it for cron.
func runFor(services *core.Services, schedule *core.Schedule) (core.Runnable, error) {
	if schedule.RefType == "call" {
		call, err := services.Calls.Get(schedule.RefGUID)
		if err != nil {
			return nil, err
		}

		return core.NewCallRun(call, schedule, services), nil
	}

	job, err := services.Jobs.Get(schedule.RefGUID)
	if err != nil {
		return nil, err
	}

	return core.NewJobRun(job, schedule, services), nil
}
