package workflows

import (
	"fmt"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// PausingAnApp disables every enabled schedule on the app's jobs and calls,
// marks each with core.PausedByUnbind, and takes it out of cron. Jobs, calls,
// schedules, and history all stay. It returns how many schedules it paused.
func PausingAnApp(services *core.Services, appGUID string) (int, error) {
	tag := "pausing-an-app"

	paused, err := services.Schedules.Pause(appGUID, core.PausedByUnbind)
	if err != nil {
		doh := fmt.Sprintf("could not pause the schedules of app %s: %v", appGUID, err)
		services.Logger.Error(tag, doh)
		return 0, fmt.Errorf("%s", doh)
	}

	pausedNow := make(map[string]bool, len(paused))
	for _, schedule := range paused {
		pausedNow[schedule.GUID] = true

		// A schedule that never loaded into cron has nothing to remove.
		if err := unschedule(services, schedule); err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("paused schedule %s but could not take it out of cron: %v", schedule.GUID, err))
		}
	}

	// Also take the app's other disabled schedules out of cron. The create
	// routes put a schedule in cron even when it was created disabled, and it
	// must not keep firing once the app is unbound. Most are not in cron, so
	// a failure here says nothing.
	for _, schedule := range services.Schedules.ByApp(appGUID) {
		if !schedule.Enabled && !pausedNow[schedule.GUID] {
			_ = unschedule(services, schedule)
		}
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

// unschedule takes a schedule out of cron.
func unschedule(services *core.Services, schedule *core.Schedule) error {
	run, err := runFor(services, schedule)
	if err != nil {
		return fmt.Errorf("could not load its %s: %w", schedule.RefType, err)
	}

	return services.Cron.Delete(run)
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
