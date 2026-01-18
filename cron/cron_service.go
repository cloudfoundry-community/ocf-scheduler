package cron

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	cron "github.com/netresearch/go-cron"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// type TimezoneSlice []Timezone
var TimezonesSlice core.TimezoneSlice = make(core.TimezoneSlice, 0, 800)

type TimezoneMap map[string]core.Timezone

var TimezonesMap TimezoneMap = make(TimezoneMap, 800)

type TimezoneFileState struct {
	Filename string
	ModTime  time.Time
	FileSize int64
}

var TimezoneFile TimezoneFileState

func (tzs *TimezoneFileState) IsModified() (bool, error) {
	if tzs.Filename == "" {
		return true, nil
	}
	stat, err := os.Stat(tzs.Filename)
	if err != nil {
		return false, err
	}
	return !stat.ModTime().Equal(tzs.ModTime) || tzs.FileSize != stat.Size(), nil
}

func (tzs *TimezoneFileState) SetState() error {
	if tzs.Filename != "" {
		stat, err := os.Stat(tzs.Filename)
		if err != nil {
			return err
		}
		tzs.ModTime = stat.ModTime()
		tzs.FileSize = stat.Size()
		return nil
	}
	return fmt.Errorf("timezone file not set")
}

type CronService struct {
	*cron.Cron
	log     core.LogService
	mapping map[string]cron.EntryID
}

func InitializeTimezones(file string) error {

	stat, err := os.Stat(file)
	if err != nil {
		return err
	}

	fileData, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(fileData, &TimezonesSlice)
	if err != nil {
		return err
	}

	TimezoneFile.Filename = file
	TimezoneFile.ModTime = stat.ModTime()
	TimezoneFile.FileSize = stat.Size()
	return nil
}

func NewCronService(log core.LogService) *CronService {
	return &CronService{
		Cron:    cron.New(cron.WithParser(cron.FullParser())),
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
	service.logMappingSize("Added job to cron service")

	return nil
}

func (service *CronService) Delete(runnable core.Runnable) error {
	id, found := service.mapping[runnable.Schedule().GUID]
	if !found {
		return fmt.Errorf("no such runner")
	}

	service.Remove(id)
	delete(service.mapping, runnable.Schedule().GUID)
	service.logMappingSize("Deleted job from cron service")

	return nil
}

func (service *CronService) Count() int {
	return len(service.Entries())
}

func (service *CronService) Validate(expression string) error {
	_, err := cron.FullParser().Parse(expression)

	return err
}

func (service *CronService) MappingSize() int {
	return len(service.mapping)
}

func (service *CronService) logMappingSize(action string) {
	size := service.MappingSize()
	service.log.Info("cron-service", fmt.Sprintf("%s: current mapping size is %d", action, size))
}

func (service *CronService) GetTimezones() (*core.TimezoneSlice, error) {
	return &TimezonesSlice, nil
}
