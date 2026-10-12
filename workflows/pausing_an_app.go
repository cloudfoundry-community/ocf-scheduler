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
//
// It puts each schedule in cron before it clears the mark, and it clears the
// mark only on the schedules that made it into cron. A schedule whose job or
// call cannot be loaded keeps its mark and stays disabled, and the error that
// follows lets Cloud Controller retry, so the retry finds it again.
func ResumingAnApp(services *core.Services, appGUID string) (int, error) {
	tag := "resuming-an-app"

	schedules, err := services.Schedules.ByApp(appGUID)
	if err != nil {
		doh := fmt.Sprintf("could not list the schedules of app %s to resume them: %v", appGUID, err)
		services.Logger.Error(tag, doh)
		return 0, fmt.Errorf("%s", doh)
	}

	failed := 0
	scheduled := make([]*core.Schedule, 0)
	for _, schedule := range schedules {
		if schedule.PausedBy != core.PausedByUnbind {
			continue
		}

		run, err := runFor(services, schedule)
		if err != nil {
			failed++
			services.Logger.Error(tag, fmt.Sprintf("could not load the %s of schedule %s to resume it: %v", schedule.RefType, schedule.GUID, err))
			continue
		}

		if err := services.Cron.Add(run); err != nil {
			failed++
			services.Logger.Error(tag, fmt.Sprintf("could not put schedule %s back in cron: %v", schedule.GUID, err))
			continue
		}

		scheduled = append(scheduled, schedule)
	}

	resumed := 0
	if len(scheduled) > 0 {
		guids := make([]string, 0, len(scheduled))
		for _, schedule := range scheduled {
			guids = append(guids, schedule.GUID)
		}

		changed, err := services.Schedules.Resume(appGUID, core.PausedByUnbind, guids)
		if err != nil {
			// The schedules are still disabled and marked, so take them back
			// out of cron for the retry to start over.
			for _, schedule := range scheduled {
				_ = unschedule(services, schedule)
			}

			doh := fmt.Sprintf("could not resume the schedules of app %s: %v", appGUID, err)
			services.Logger.Error(tag, doh)
			return 0, fmt.Errorf("%s", doh)
		}

		// A schedule the update did not change was deleted after ByApp
		// listed it, so it must not keep its cron entry.
		kept := make(map[string]bool, len(changed))
		for _, schedule := range changed {
			kept[schedule.GUID] = true
		}
		for _, schedule := range scheduled {
			if !kept[schedule.GUID] {
				_ = unschedule(services, schedule)
			}
		}

		resumed = len(changed)
	}

	if failed > 0 {
		return resumed, fmt.Errorf("resumed %d schedules of app %s but could not put %d back in cron", resumed, appGUID, failed)
	}

	services.Logger.Info(tag, fmt.Sprintf("resumed %d schedules for app %s", resumed, appGUID))

	return resumed, nil
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
