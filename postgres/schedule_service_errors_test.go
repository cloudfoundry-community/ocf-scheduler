package postgres

import (
	"database/sql"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
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
