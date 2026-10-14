package routes

import (
	"errors"
	"net/http"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// lateJobs and lateCalls run late just before they delete, standing in for
// a schedule create that lands after a delete has listed the schedules it
// will remove.
type lateJobs struct {
	core.JobService
	late func(core.Executable)
}

func (jobs lateJobs) Delete(job *core.Job) error {
	jobs.late(job)
	return jobs.JobService.Delete(job)
}

type lateCalls struct {
	core.CallService
	late func(core.Executable)
}

func (calls lateCalls) Delete(call *core.Call) error {
	calls.late(call)
	return calls.CallService.Delete(call)
}

// expectScheduleGone checks that a schedule is out of both the database and
// cron.
func expectScheduleGone(t *testing.T, tn *tenancy, schedule *core.Schedule, executable core.Executable) {
	t.Helper()
	if _, err := tn.services.Schedules.Get(schedule.GUID); err == nil {
		t.Errorf("schedule %s is still stored", schedule.GUID)
	}
	expectUnscheduled(t, tn, schedule, executable)
}

// expectUnscheduled checks that a schedule is out of cron.
func expectUnscheduled(t *testing.T, tn *tenancy, schedule *core.Schedule, executable core.Executable) {
	t.Helper()
	var run core.Runnable
	if job, err := executable.ToJob(); err == nil {
		run = core.NewJobRun(job, schedule, tn.services)
	} else {
		call, _ := executable.ToCall()
		run = core.NewCallRun(call, schedule, tn.services)
	}
	if err := tn.services.Cron.Delete(run); !errors.Is(err, core.ErrNotScheduled) {
		t.Errorf("schedule %s is still in cron", schedule.GUID)
	}
}

// A schedule created while a job or call is being deleted is removed with
// it, whichever route deletes it.
func TestDeletesRemoveAScheduleCreatedDuringTheDelete(t *testing.T) {
	for _, tt := range []struct {
		name   string
		target func(tn *tenancy) core.Executable
		delete func(t *testing.T, tn *tenancy, guid string)
	}{
		{
			name:   "orphan job",
			target: func(tn *tenancy) core.Executable { return tn.jobB },
			delete: func(t *testing.T, tn *tenancy, guid string) {
				expectResult(t, tn.deleteOrphans(t, guid), guid, "job", orphanDeleted)
			},
		},
		{
			name:   "orphan call",
			target: func(tn *tenancy) core.Executable { return tn.callB },
			delete: func(t *testing.T, tn *tenancy, guid string) {
				expectResult(t, tn.deleteOrphans(t, guid), guid, "call", orphanDeleted)
			},
		},
		{
			name:   "job route",
			target: func(tn *tenancy) core.Executable { return tn.jobB },
			delete: func(t *testing.T, tn *tenancy, guid string) {
				if rec := tn.do(http.MethodDelete, "/jobs/"+guid, "", tn.userB); rec.Code != http.StatusNoContent {
					t.Errorf("status %d, body %s", rec.Code, rec.Body)
				}
			},
		},
		{
			name:   "call route",
			target: func(tn *tenancy) core.Executable { return tn.callB },
			delete: func(t *testing.T, tn *tenancy, guid string) {
				if rec := tn.do(http.MethodDelete, "/calls/"+guid, "", tn.userB); rec.Code != http.StatusNoContent {
					t.Errorf("status %d, body %s", rec.Code, rec.Body)
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tn := newTenancy(t)
			tn.cf.RemoveApp("app-b")
			target := tt.target(tn)

			var late *core.Schedule
			lateOnce := func(executable core.Executable) {
				if late == nil {
					late = tn.persistSchedule(t, executable)
				}
			}
			tn.services.Jobs = lateJobs{tn.services.Jobs, lateOnce}
			tn.services.Calls = lateCalls{tn.services.Calls, lateOnce}

			tt.delete(t, tn, ownerOf(target).GUID)

			if late == nil {
				t.Fatal("the delete never reached the stored job or call")
			}
			expectScheduleGone(t, tn, late, target)
		})
	}
}

// vanishingSchedules deletes the schedule's job or call just before the
// schedule is stored, standing in for a delete that finishes, sweep and
// all, between the create's lookup and its write.
type vanishingSchedules struct {
	core.ScheduleService
	tn *tenancy
}

func (schedules vanishingSchedules) Persist(schedule *core.Schedule) (*core.Schedule, error) {
	if schedule.RefType == "job" {
		schedules.tn.services.Jobs.Delete(&core.Job{GUID: schedule.RefGUID})
	} else {
		schedules.tn.services.Calls.Delete(&core.Call{GUID: schedule.RefGUID})
	}

	return schedules.ScheduleService.Persist(schedule)
}

// A create whose job or call is deleted before the schedule is stored takes
// the schedule back out and answers not found, rather than leaving a
// schedule that fires for something that no longer exists.
func TestCreateScheduleUndoesAScheduleWhoseOwnerWentAway(t *testing.T) {
	for _, tt := range []struct {
		name   string
		target func(tn *tenancy) core.Executable
	}{
		{name: "job", target: func(tn *tenancy) core.Executable { return tn.jobB }},
		{name: "call", target: func(tn *tenancy) core.Executable { return tn.callB }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tn := newTenancy(t)
			target := tt.target(tn)
			guid := ownerOf(target).GUID
			before := tn.services.Cron.Count()

			var stored *core.Schedule
			tn.services.Schedules = &capturingSchedules{vanishingSchedules{tn.services.Schedules, tn}, &stored}

			rec := tn.do(http.MethodPost, "/"+tt.name+"s/"+guid+"/schedules",
				`{"enabled": true, "expression": "0 3 * * *", "expression_type": "cron_expression"}`, tn.userB)
			if rec.Code != http.StatusNotFound {
				t.Errorf("status %d, want %d, body %s", rec.Code, http.StatusNotFound, rec.Body)
			}

			if stored == nil {
				t.Fatal("the create never stored a schedule")
			}
			expectScheduleGone(t, tn, stored, target)
			if n := tn.services.Cron.Count(); n != before {
				t.Errorf("cron holds %d entries, want %d", n, before)
			}
		})
	}
}

// capturingSchedules remembers the last schedule stored through it.
type capturingSchedules struct {
	core.ScheduleService
	stored **core.Schedule
}

func (schedules *capturingSchedules) Persist(schedule *core.Schedule) (*core.Schedule, error) {
	stored, err := schedules.ScheduleService.Persist(schedule)
	*schedules.stored = stored
	return stored, err
}

// stuckSchedules fails every schedule delete once it is armed, standing in
// for a database that drops the connection partway through.
type stuckSchedules struct {
	core.ScheduleService
	armed *bool
}

func (schedules stuckSchedules) Delete(schedule *core.Schedule) error {
	if *schedules.armed {
		return errors.New("database unavailable")
	}
	return schedules.ScheduleService.Delete(schedule)
}

// A schedule whose job or call is gone comes out of cron even when its row
// cannot be deleted, because the entry would otherwise go on running a job
// or call that no longer exists.
func TestAScheduleWhoseOwnerIsGoneLeavesCronWhenItsRowStays(t *testing.T) {
	for _, tt := range []struct {
		name   string
		target func(tn *tenancy) core.Executable
		act    func(t *testing.T, tn *tenancy, target core.Executable, armed *bool) *core.Schedule
	}{
		{
			name:   "create undo",
			target: func(tn *tenancy) core.Executable { return tn.jobB },
			act: func(t *testing.T, tn *tenancy, target core.Executable, armed *bool) *core.Schedule {
				var stored *core.Schedule
				tn.services.Schedules = &capturingSchedules{vanishingSchedules{tn.services.Schedules, tn}, &stored}
				*armed = true
				tn.services.Schedules = stuckSchedules{tn.services.Schedules, armed}

				rec := tn.do(http.MethodPost, "/jobs/"+ownerOf(target).GUID+"/schedules",
					`{"enabled": true, "expression": "0 3 * * *", "expression_type": "cron_expression"}`, tn.userB)
				if rec.Code != http.StatusNotFound {
					t.Errorf("status %d, want %d, body %s", rec.Code, http.StatusNotFound, rec.Body)
				}
				return stored
			},
		},
		{
			name:   "delete sweep",
			target: func(tn *tenancy) core.Executable { return tn.callB },
			act: func(t *testing.T, tn *tenancy, target core.Executable, armed *bool) *core.Schedule {
				var late *core.Schedule
				tn.services.Calls = lateCalls{tn.services.Calls, func(executable core.Executable) {
					if late == nil {
						late = tn.persistSchedule(t, executable)
						*armed = true
					}
				}}

				if rec := tn.do(http.MethodDelete, "/calls/"+ownerOf(target).GUID, "", tn.userB); rec.Code != http.StatusNoContent {
					t.Errorf("status %d, body %s", rec.Code, rec.Body)
				}
				return late
			},
		},
		{
			name:   "orphan schedule delete",
			target: func(tn *tenancy) core.Executable { return &core.Job{GUID: "gone-job"} },
			act: func(t *testing.T, tn *tenancy, target core.Executable, armed *bool) *core.Schedule {
				schedule := tn.persistSchedule(t, target)
				*armed = true

				expectResult(t, tn.deleteOrphans(t, schedule.GUID), schedule.GUID, "schedule", orphanFailed)
				return schedule
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tn := newTenancy(t)
			target := tt.target(tn)
			armed := false
			tn.services.Schedules = stuckSchedules{tn.services.Schedules, &armed}

			schedule := tt.act(t, tn, target, &armed)
			if schedule == nil {
				t.Fatal("no schedule was stored")
			}

			expectUnscheduled(t, tn, schedule, target)
		})
	}
}

// brokenJobs and brokenCalls fail every lookup with an error other than
// not found.
type brokenJobs struct{ core.JobService }

func (brokenJobs) Get(string) (*core.Job, error) {
	return nil, errors.New("database unavailable")
}

type brokenCalls struct{ core.CallService }

func (brokenCalls) Get(string) (*core.Call, error) {
	return nil, errors.New("database unavailable")
}

// breakingSchedules breaks job and call lookups just as the schedule is
// stored, so only the create's second lookup fails.
type breakingSchedules struct {
	core.ScheduleService
	tn *tenancy
}

func (schedules breakingSchedules) Persist(schedule *core.Schedule) (*core.Schedule, error) {
	schedules.tn.services.Jobs = brokenJobs{schedules.tn.services.Jobs}
	schedules.tn.services.Calls = brokenCalls{schedules.tn.services.Calls}
	return schedules.ScheduleService.Persist(schedule)
}

// A create whose second lookup of the job or call fails keeps the schedule
// and answers created. The owner was there a moment before, and an error
// would invite a retry that stores a second schedule.
func TestCreateScheduleKeepsTheScheduleWhenTheOwnerCannotBeLookedUpAgain(t *testing.T) {
	for _, tt := range []struct {
		name   string
		target func(tn *tenancy) core.Executable
	}{
		{name: "job", target: func(tn *tenancy) core.Executable { return tn.jobB }},
		{name: "call", target: func(tn *tenancy) core.Executable { return tn.callB }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tn := newTenancy(t)
			target := tt.target(tn)
			before := tn.services.Cron.Count()

			var stored *core.Schedule
			tn.services.Schedules = &capturingSchedules{breakingSchedules{tn.services.Schedules, tn}, &stored}

			rec := tn.do(http.MethodPost, "/"+tt.name+"s/"+ownerOf(target).GUID+"/schedules",
				`{"enabled": true, "expression": "0 3 * * *", "expression_type": "cron_expression"}`, tn.userB)
			if rec.Code != http.StatusCreated {
				t.Errorf("status %d, want %d, body %s", rec.Code, http.StatusCreated, rec.Body)
			}

			if stored == nil {
				t.Fatal("the create never stored a schedule")
			}
			if _, err := tn.services.Schedules.Get(stored.GUID); err != nil {
				t.Errorf("schedule %s is no longer stored: %v", stored.GUID, err)
			}
			if n := tn.services.Cron.Count(); n != before+1 {
				t.Errorf("cron holds %d entries, want %d", n, before+1)
			}
		})
	}
}
