package cron

import (
	"encoding/json"
	"errors"
	"fmt"
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
	rules   Rules
	mapping map[string]cron.EntryID
}

func GetServerTimezone() (string, error) {
	for _, tz := range TimezonesSlice {
		if tz.IsServerTimeZone != "" {
			return tz.Name, nil
		}
	}
	return "", errors.New("No server timezone found")
}

func InitializeTimezones(file string) error {

	stat, err := os.Stat(file)
	if err != nil {
		return err
	}

	fileData, err := os.ReadFile(file)
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

func NewCronService(log core.LogService, rules Rules) *CronService {
	return &CronService{
		Cron:    cron.New(cron.WithParser(cron.FullParser())),
		log:     log,
		rules:   rules,
		mapping: make(map[string]cron.EntryID),
	}
}

// Add registers the runnable's schedule. Only a parse failure stops it:
// a stored schedule that breaks a later policy rule (never fires, under the
// minimum interval) still loads, with a warning, so raising a limit never
// silently stops existing schedules. Create rejects those before Add.
func (service *CronService) Add(runnable core.Runnable) error {
	schedule := runnable.Schedule()
	analysis := Analyze(schedule.Expression, schedule.RefGUID, 0, 0, service.rules, time.Now())
	if analysis.Schedule == nil {
		return fmt.Errorf("invalid cron expression %q: %s", schedule.Expression, analysis.Result.Errors[0].Message)
	}
	for _, finding := range analysis.Result.Errors {
		service.log.Warn("cron-service", fmt.Sprintf("schedule %s (%s): %s", schedule.GUID, schedule.Expression, finding.Message))
	}

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

	id, err := service.ScheduleJob(analysis.Schedule, cron.FuncJob(process))
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

// Analyze applies this service's rules to expression, keyed by key (the job
// or call GUID; empty for a bare expression). Runs are listed from from;
// the zero time means now.
func (service *CronService) Analyze(expression, key string, next, prev int, from time.Time) core.ScheduleAnalysis {
	if from.IsZero() {
		from = time.Now()
	}
	result := Analyze(expression, key, next, prev, service.rules, from).Result
	if result.Location == "Local" {
		if name, err := GetServerTimezone(); err == nil {
			result.Location = name
		}
	}
	return result
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
