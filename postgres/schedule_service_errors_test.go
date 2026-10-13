package postgres

import (
	"database/sql"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/ess/testscope"
)

// lostDatabase returns a service whose every query fails, as a lost database
// would.
func lostDatabase(t *testing.T) *ScheduleService {
	t.Helper()

	db, err := sql.Open("postgres", "postgres://nobody@127.0.0.1:1/none?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	return NewScheduleService(db)
}

func TestScheduleService_ByJobReportsAFailedQuery(t *testing.T) {
	found, err := lostDatabase(t).ByJob(&core.Job{GUID: "job-1"})

	if err == nil {
		t.Errorf("expected an error, got %d schedules", len(found))
	}
}

func TestScheduleService_ByCallReportsAFailedQuery(t *testing.T) {
	found, err := lostDatabase(t).ByCall(&core.Call{GUID: "call-1"})

	if err == nil {
		t.Errorf("expected an error, got %d schedules", len(found))
	}
}

func TestScheduleService_DeleteReportsAFailedLookup(t *testing.T) {
	if err := lostDatabase(t).Delete(&core.Schedule{GUID: "schedule-1"}); err == nil {
		t.Errorf("expected an error, the schedule was not deleted")
	}
}

func TestScheduleService_DeleteOfAMissingScheduleSucceeds(t *testing.T) {
	testscope.SkipUnlessUnit(t)

	guid, _ := core.GenGUID()

	if err := NewScheduleService(testdb).Delete(&core.Schedule{GUID: guid}); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
