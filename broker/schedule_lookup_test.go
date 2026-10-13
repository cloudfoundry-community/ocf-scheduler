package broker_test

import (
	"fmt"
	nethttp "net/http"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

// failingLookups fails every by-job and by-call lookup, as a lost database
// would.
type failingLookups struct {
	*mock.ScheduleService
}

func (failingLookups) ByJob(*core.Job) ([]*core.Schedule, error) {
	return nil, fmt.Errorf("%s", databaseError)
}

func (failingLookups) ByCall(*core.Call) ([]*core.Schedule, error) {
	return nil, fmt.Errorf("%s", databaseError)
}

// A job or call delete whose schedule lookup fails answers 500 and changes
// nothing, neither the rows nor cron.
func TestDeletingAnOwnerChangesNothingWhenTheScheduleLookupFails(t *testing.T) {
	f := newFixture(t)
	f.services.Auth.(*mock.AuthService).Developer("jeremy", "space-a")
	f.services.Schedules = failingLookups{f.services.Schedules.(*mock.ScheduleService)}

	expectStatus(t, "delete job",
		f.do(nethttp.MethodDelete, "/jobs/"+f.job.GUID, "", withToken("jeremy")),
		nethttp.StatusInternalServerError)
	expectStatus(t, "delete call",
		f.do(nethttp.MethodDelete, "/calls/"+f.call.GUID, "", withToken("jeremy")),
		nethttp.StatusInternalServerError)

	if _, err := f.services.Jobs.Get(f.job.GUID); err != nil {
		t.Errorf("the job must stay stored, got %v", err)
	}
	if _, err := f.services.Calls.Get(f.call.GUID); err != nil {
		t.Errorf("the call must stay stored, got %v", err)
	}

	for name, schedule := range map[string]*core.Schedule{"nightly": f.nightly, "hourly": f.hourly, "ping": f.ping} {
		if _, err := f.services.Schedules.Get(schedule.GUID); err != nil {
			t.Errorf("%s must stay stored, got %v", name, err)
		}
		if !f.cron.Scheduled(schedule.GUID) {
			t.Errorf("%s must stay in cron", name)
		}
	}
}
