package postgres

import (
	"database/sql"
	"errors"
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

func TestCallService_DeleteReportsAFailedLookup(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://nobody@127.0.0.1:1/none?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	if err := NewCallService(db).Delete(&core.Call{GUID: "call-1"}); err == nil {
		t.Errorf("expected an error, the call was not deleted")
	}
}

func TestJobService_DeleteOfAMissingJobWrapsNotFound(t *testing.T) {
	testscope.SkipUnlessUnit(t)

	guid, _ := core.GenGUID()

	err := NewJobService(testdb).Delete(&core.Job{GUID: guid})
	if !errors.Is(err, core.ErrNotFound) {
		t.Errorf("expected an error that wraps core.ErrNotFound, got %v", err)
	}
}

func TestScheduleService_DeleteReportsARowThatFailsToScan(t *testing.T) {
	testscope.SkipUnlessUnit(t)

	guid, _ := core.GenGUID()
	refGUID, _ := core.GenGUID()

	Cleaner.Acquire("schedules")
	defer Cleaner.Clean("schedules")

	// enabled is nullable, so a row edited by hand can hold a value that the
	// scan cannot read.
	_, err := testdb.Exec(
		"INSERT INTO schedules (guid, enabled, expression, expression_type, ref_guid, ref_type, created_at, updated_at) VALUES ($1, NULL, '* * * * *', 'cron_expression', $2, 'job', now(), now())",
		guid, refGUID,
	)
	if err != nil {
		t.Fatal(err)
	}

	service := NewScheduleService(testdb)

	if err := service.Delete(&core.Schedule{GUID: guid}); err == nil {
		t.Errorf("expected an error, the row was reported as missing")
	}

	if _, err := service.ByJob(&core.Job{GUID: refGUID}); err == nil {
		t.Errorf("expected an error, the row was left out of the list")
	}
}

// A lookup that fails is not reported as a schedule that does not exist.
func TestScheduleService_GetReportsAFailedQuery(t *testing.T) {
	found, err := lostDatabase(t).Get("some-guid")

	if err == nil {
		t.Fatalf("expected an error, got %v", found)
	}
	if errors.Is(err, core.ErrNotFound) {
		t.Errorf("a failed query reads as not found: %v", err)
	}
}

func TestScheduleService_OrphanedReportsAFailedQuery(t *testing.T) {
	found, err := lostDatabase(t).Orphaned()

	if err == nil {
		t.Errorf("expected an error, got %d schedules", len(found))
	}
}

// Orphaned finds the schedules whose job or call is gone, including one
// that names a call as its job, and leaves out the ones whose owner exists.
func TestScheduleService_OrphanedFindsSchedulesWhoseOwnerIsGone(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	Cleaner.Acquire("jobs", "calls", "schedules")
	defer Cleaner.Clean("jobs", "calls", "schedules")

	jobs := NewJobService(testdb)
	calls := NewCallService(testdb)
	schedules := NewScheduleService(testdb)

	app, _ := core.GenGUID()
	space, _ := core.GenGUID()
	job, err := jobs.Persist(&core.Job{Name: "backup", Command: "true", AppGUID: app, SpaceGUID: space})
	if err != nil {
		t.Fatal(err)
	}
	call, err := calls.Persist(&core.Call{Name: "ping", URL: "https://example.com", AuthHeader: "x", AppGUID: app, SpaceGUID: space})
	if err != nil {
		t.Fatal(err)
	}
	goneJob, _ := core.GenGUID()
	goneCall, _ := core.GenGUID()

	persist := func(refType, refGUID string) *core.Schedule {
		t.Helper()
		schedule, err := schedules.Persist(&core.Schedule{Enabled: true, Expression: "0 2 * * *", ExpressionType: "cron_expression", RefType: refType, RefGUID: refGUID})
		if err != nil {
			t.Fatal(err)
		}
		return schedule
	}

	persist("job", job.GUID)
	persist("call", call.GUID)
	want := map[string]bool{
		persist("job", goneJob).GUID:   true,
		persist("call", goneCall).GUID: true,
		persist("job", call.GUID).GUID: true,
	}

	found, err := schedules.Orphaned()
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, schedule := range found {
		got[schedule.GUID] = true
	}
	if len(found) != len(want) || len(got) != len(want) {
		t.Errorf("got %d orphaned schedules, want %d", len(found), len(want))
	}
	for guid := range want {
		if !got[guid] {
			t.Errorf("schedule %s is not listed as orphaned", guid)
		}
	}
}
