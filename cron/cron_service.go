package cron

import (
	"fmt"

	"github.com/robfig/cron/v3"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

type CronService struct {
	*cron.Cron
	log     core.LogService
	mapping map[string]cron.EntryID
}

func NewCronService(log core.LogService) *CronService {
	return &CronService{
		Cron:    cron.New(),
		log:     log,
		mapping: make(map[string]cron.EntryID),
	}
}

func (service *CronService) Add(runnable core.Runnable) error {
	schedule := runnable.Schedule()

	process := func() {
		tag := "cron-service"

		if runnable.Job() != nil {
			job := runnable.Job()

			message := fmt.Sprintf(
				"running job %s (%s) [%s]",
				job.Name,
				job.Command,
				schedule.Expression,
			)

			service.log.Info(tag, message)
		} else {
			call := runnable.Call()

			message := fmt.Sprintf(
				"performing call %s (%s) [%s]",
				call.Name,
				call.URL,
				schedule.Expression,
			)

			service.log.Info(tag, message)
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

func (service *CronService) Validate(expression string) error {
	_, err := cron.ParseStandard(expression)

	return err
}
