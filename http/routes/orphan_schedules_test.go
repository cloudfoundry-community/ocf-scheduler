package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// listedSchedule is an orphan schedule as the list sends it.
type listedSchedule struct {
	GUID       string `json:"guid"`
	Enabled    bool   `json:"enabled"`
	Expression string `json:"expression"`
	RefType    string `json:"ref_type"`
	RefGUID    string `json:"ref_guid"`
}

func (tn *tenancy) listOrphanSchedules(t *testing.T) []*listedSchedule {
	t.Helper()
	rec := tn.do(http.MethodGet, "/orphans", "", tn.admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: status %d, body %s", rec.Code, rec.Body)
	}
	var output struct {
		Resources       []*orphan         `json:"resources"`
		OrphanSchedules []*listedSchedule `json:"orphan_schedules"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &output); err != nil {
		t.Fatalf("list: %v, body %s", err, rec.Body)
	}
	if output.OrphanSchedules == nil {
		t.Errorf("list: no orphan_schedules list in %s", rec.Body)
	}
	return output.OrphanSchedules
}

// persistOrphanSchedules stores, and adds to cron, a schedule for a job and
// one for a call that are not stored, and returns them with their stand-in
// owners.
func (tn *tenancy) persistOrphanSchedules(t *testing.T) ([]*core.Schedule, []core.Executable) {
	t.Helper()
	goneJob, _ := core.GenGUID()
	goneCall, _ := core.GenGUID()
	owners := []core.Executable{&core.Job{GUID: goneJob}, &core.Call{GUID: goneCall}}

	schedules := make([]*core.Schedule, 0, len(owners))
	for _, owner := range owners {
		schedules = append(schedules, tn.persistSchedule(t, owner))
	}

	return schedules, owners
}

// The list names each schedule whose job or call is gone, even when every
// app exists, and leaves out the schedules whose job or call is stored.
func TestListOrphansShowsSchedulesWhoseOwnerIsGone(t *testing.T) {
	tn := newTenancy(t)

	if listed := tn.listOrphanSchedules(t); len(listed) != 0 {
		t.Errorf("listed %d orphan schedules before any, want none", len(listed))
	}

	orphans, _ := tn.persistOrphanSchedules(t)

	listed := tn.listOrphanSchedules(t)
	if len(listed) != 2 {
		t.Fatalf("listed %d orphan schedules, want 2", len(listed))
	}
	// The list is sorted by type, so the call's schedule comes first.
	for i, want := range []*core.Schedule{orphans[1], orphans[0]} {
		got := listed[i]
		if got.GUID != want.GUID || got.RefType != want.RefType || got.RefGUID != want.RefGUID || got.Expression != want.Expression || !got.Enabled {
			t.Errorf("orphan schedule %d is %+v, want %+v", i, got, want)
		}
	}

	if n := tn.services.Cron.Count(); n != 4 {
		t.Errorf("cron holds %d entries after the list, want 4", n)
	}
}

// failingOrphaned fails the orphan schedule query, as a lost database would.
type failingOrphaned struct {
	core.ScheduleService
}

func (failingOrphaned) Orphaned() ([]*core.Schedule, error) {
	return nil, errors.New("database unavailable")
}

func TestListOrphansFailsWhenTheScheduleQueryFails(t *testing.T) {
	tn := newTenancy(t)
	tn.services.Schedules = failingOrphaned{tn.services.Schedules}

	if rec := tn.do(http.MethodGet, "/orphans", "", tn.admin); rec.Code != http.StatusInternalServerError {
		t.Errorf("status %d, want %d, body %s", rec.Code, http.StatusInternalServerError, rec.Body)
	}
}

// The delete takes an orphan schedule's GUID and removes it from the
// database and cron, but refuses a schedule whose job or call is stored.
func TestDeleteOrphansDeletesSchedulesWhoseOwnerIsGone(t *testing.T) {
	tn := newTenancy(t)
	orphans, owners := tn.persistOrphanSchedules(t)

	results := tn.deleteOrphans(t, orphans[0].GUID, orphans[1].GUID, tn.jobSchedB.GUID, tn.callSchedB.GUID)

	expectResult(t, results, orphans[0].GUID, "schedule", "deleted")
	expectResult(t, results, orphans[1].GUID, "schedule", "deleted")
	expectResult(t, results, tn.jobSchedB.GUID, "schedule", "parent_exists")
	expectResult(t, results, tn.callSchedB.GUID, "schedule", "parent_exists")

	for i, schedule := range orphans {
		expectScheduleGone(t, tn, schedule, owners[i])
	}
	for _, schedule := range []*core.Schedule{tn.jobSchedB, tn.callSchedB} {
		if _, err := tn.services.Schedules.Get(schedule.GUID); err != nil {
			t.Errorf("schedule %s was deleted, but its owner exists", schedule.GUID)
		}
	}
	tn.assertBUntouched(t)
	if n := tn.services.Cron.Count(); n != 2 {
		t.Errorf("cron holds %d entries, want 2", n)
	}

	// Deleting them again finds nothing.
	results = tn.deleteOrphans(t, orphans[0].GUID)
	expectResult(t, results, orphans[0].GUID, "", "not_found")
}

// Orphan schedules never send a request to Cloud Controller, so they can be
// listed and deleted when the scheduler's client lacks the app reader scope.
// A request that names a job or call is still refused.
func TestOrphanSchedulesNeedNoAppReaderScope(t *testing.T) {
	tn := newTenancy(t)
	orphans, _ := tn.persistOrphanSchedules(t)
	tn.cf.SetClientScopes("cloud_controller.read")

	// With no jobs or calls to check, the list asks about no app.
	jobs, calls := tn.services.Jobs, tn.services.Calls
	tn.services.Jobs, tn.services.Calls = noJobs{jobs}, noCalls{calls}
	if listed := tn.listOrphanSchedules(t); len(listed) != 2 {
		t.Errorf("listed %d orphan schedules, want 2", len(listed))
	}
	tn.services.Jobs, tn.services.Calls = jobs, calls

	expectResult(t, tn.deleteOrphans(t, orphans[0].GUID), orphans[0].GUID, "schedule", orphanDeleted)

	body := `{"guids": ["` + orphans[1].GUID + `", "` + tn.jobB.GUID + `"]}`
	if rec := tn.do(http.MethodPost, "/orphans/delete", body, tn.admin); rec.Code != http.StatusInternalServerError {
		t.Errorf("a delete naming a job: status %d, want %d, body %s", rec.Code, http.StatusInternalServerError, rec.Body)
	}
}

// noJobs and noCalls list nothing, though their lookups still work.
type noJobs struct{ core.JobService }

func (noJobs) All() ([]*core.Job, error) { return nil, nil }

type noCalls struct{ core.CallService }

func (noCalls) All() ([]*core.Call, error) { return nil, nil }
