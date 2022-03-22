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

func (service *CronService) Add(runner *core.Run) error {
	process := func() {
		fmt.Printf(
			"Running job %s (%s)\n\tScheduled with %s (%s)\n",
			runner.Job.Name,
			runner.Job.Command,
			runner.Schedule.Expression,
			runner.Schedule.GUID,
		)

		runner.Run()
	}

	id, err := service.AddFunc(runner.Schedule.Expression, process)
	if err != nil {
		return err
	}

	service.mapping[runner.Schedule.GUID] = id

	return nil
}

func (service *CronService) Delete(runner *core.Run) error {
	id, found := service.mapping[runner.Schedule.GUID]
	if !found {
		return fmt.Errorf("no such runner")
	}

	service.Remove(id)

	return nil
}

func (service *CronService) Count() int {
	return len(service.Entries())
}
