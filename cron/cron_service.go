package cron

import (
	"fmt"

	"github.com/robfig/cron/v3"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

type CronService struct {
	*cron.Cron
	mapping map[string]cron.EntryID
}

func NewCronService() *CronService {
	return &CronService{cron.New(), make(map[string]cron.EntryID)}
}

func (service *CronService) Add(runnable core.Runnable) error {
	schedule := runnable.Schedule()

	process := func() {
		if runnable.Job() != nil {
			job := runnable.Job()

			fmt.Printf(
				"\nRunning job %s (%s)\n\tScheduled with %s (%s)\n",
				job.Name,
				job.Command,
				schedule.Expression,
				schedule.GUID,
			)
		} else {
			call := runnable.Call()

			fmt.Printf(
				"\nPerforming call %s (%s) [%s]\n\tScheduled with %s (%s)\n",
				call.Name,
				call.GUID,
				call.URL,
				schedule.Expression,
				schedule.GUID,
			)
		}

		runnable.Run()
	}

	id, err := service.AddFunc(schedule.Expression, process)
	if err != nil {
		return err
	}

	service.mapping[schedule.GUID] = id

	return nil
}

func (service *CronService) Delete(runnable core.Runnable) error {
	id, found := service.mapping[runnable.Schedule().GUID]
	if !found {
		return fmt.Errorf("no such runner")
	}

	service.Remove(id)

	return nil
}

func (service *CronService) Count() int {
	return len(service.Entries())
}
