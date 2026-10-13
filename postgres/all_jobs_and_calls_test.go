package postgres

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/ess/testscope"
)

// lostDB returns a database whose every query fails, as a lost database
// would.
func lostDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres", "postgres://nobody@127.0.0.1:1/none?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	return db
}

func TestJobService_AllReportsAFailedQuery(t *testing.T) {
	found, err := NewJobService(lostDB(t)).All()

	if err == nil {
		t.Errorf("expected an error, got %d jobs", len(found))
	}
}

func TestCallService_AllReportsAFailedQuery(t *testing.T) {
	found, err := NewCallService(lostDB(t)).All()

	if err == nil {
		t.Errorf("expected an error, got %d calls", len(found))
	}
}

// A call lookup that fails is not a missing call.
func TestCallService_GetReportsAFailedQuery(t *testing.T) {
	_, err := NewCallService(lostDB(t)).Get("call-1")

	if err == nil || errors.Is(err, core.ErrNotFound) {
		t.Errorf("expected a lookup error, got %v", err)
	}
}

func TestJobService_GetReportsAFailedQuery(t *testing.T) {
	_, err := NewJobService(lostDB(t)).Get("job-1")

	if err == nil || errors.Is(err, core.ErrNotFound) {
		t.Errorf("expected a lookup error, got %v", err)
	}
}

func TestJobsAndCallsAll(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	Cleaner.Acquire("jobs", "calls")
	defer Cleaner.Clean("jobs", "calls")

	jobs := NewJobService(testdb)
	calls := NewCallService(testdb)

	appA, _ := core.GenGUID()
	appB, _ := core.GenGUID()
	space, _ := core.GenGUID()

	for _, app := range []string{appA, appB} {
		if _, err := jobs.Persist(&core.Job{Name: "backup-" + app, Command: "true", AppGUID: app, SpaceGUID: space}); err != nil {
			t.Fatal(err)
		}
		if _, err := calls.Persist(&core.Call{Name: "ping-" + app, URL: "https://example.com", AuthHeader: "x", AppGUID: app, SpaceGUID: space}); err != nil {
			t.Fatal(err)
		}
	}

	allJobs, err := jobs.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(allJobs) != 2 {
		t.Errorf("got %d jobs, want 2", len(allJobs))
	}

	allCalls, err := calls.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(allCalls) != 2 {
		t.Errorf("got %d calls, want 2", len(allCalls))
	}

	seen := map[string]bool{}
	for _, call := range allCalls {
		seen[call.AppGUID] = true
		if call.AuthHeader != "x" || call.SpaceGUID != space {
			t.Errorf("call %s came back as %+v", call.GUID, call)
		}
	}
	if !seen[appA] || !seen[appB] {
		t.Errorf("calls came back for apps %v, want %s and %s", seen, appA, appB)
	}

	missing, _ := core.GenGUID()
	if _, err := jobs.Get(missing); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("a missing job: want ErrNotFound, got %v", err)
	}
	if _, err := calls.Get(missing); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("a missing call: want ErrNotFound, got %v", err)
	}
}
