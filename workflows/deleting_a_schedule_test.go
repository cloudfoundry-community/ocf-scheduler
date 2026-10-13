package workflows

import (
	"errors"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/cron"
	"github.com/cloudfoundry-community/ocf-scheduler/logger"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

// failingSchedules fails every delete, as a lost database or a row that
// cannot be read would.
type failingSchedules struct {
	core.ScheduleService
}

func (failingSchedules) Delete(*core.Schedule) error {
	return errors.New("the database is gone")
}

func TestDeletingAScheduleLeavesCronAloneWhenTheRowDeleteFails(t *testing.T) {
	log := logger.New()
	services := &core.Services{
		Jobs:      mock.NewJobService(),
		Schedules: failingSchedules{mock.NewScheduleService()},
		Cron:      cron.NewCronService(log, cron.Rules{}),
		Logger:    log,
	}

	job, err := services.Jobs.Persist(&core.Job{Name: "job", Command: "true", AppGUID: "app", SpaceGUID: "space"})
	if err != nil {
		t.Fatal(err)
	}

	schedule := &core.Schedule{GUID: "schedule-1", Enabled: true, Expression: "0 2 * * *", ExpressionType: "cron_expression", RefGUID: job.GUID, RefType: "job"}
	if err := services.Cron.Add(core.NewJobRun(job, schedule, services)); err != nil {
		t.Fatal(err)
	}

	if err := DeletingASchedule(services, schedule, job); err == nil {
		t.Fatal("expected an error, the row delete failed")
	}

	if n := services.Cron.Count(); n != 1 {
		t.Errorf("cron holds %d entries, want 1, because the cron entry must stay while the row does", n)
	}
}
