package broker_test

import (
	"fmt"
	nethttp "net/http"
	"strings"
	"sync"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/broker"
	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

const databaseError = "pq: connection refused dial tcp 10.0.0.5:5432"

// flakyJobs fails every job lookup while fail is set, as a lost database
// would.
type flakyJobs struct {
	*mock.JobService

	locker sync.Mutex
	fail   bool
}

func (jobs *flakyJobs) Get(guid string) (*core.Job, error) {
	jobs.locker.Lock()
	fail := jobs.fail
	jobs.locker.Unlock()

	if fail {
		return nil, fmt.Errorf("%s", databaseError)
	}

	return jobs.JobService.Get(guid)
}

func (jobs *flakyJobs) failing(fail bool) {
	jobs.locker.Lock()
	defer jobs.locker.Unlock()

	jobs.fail = fail
}

// flakySchedules fails Pause or Resume with the error set for it.
type flakySchedules struct {
	*mock.ScheduleService

	pauseErr, resumeErr error
}

func (schedules *flakySchedules) Pause(appGUID string, reason string) ([]*core.Schedule, error) {
	if schedules.pauseErr != nil {
		return nil, schedules.pauseErr
	}

	return schedules.ScheduleService.Pause(appGUID, reason)
}

func (schedules *flakySchedules) Resume(appGUID string, reason string) ([]*core.Schedule, error) {
	if schedules.resumeErr != nil {
		return nil, schedules.resumeErr
	}

	return schedules.ScheduleService.Resume(appGUID, reason)
}

func (f *fixture) flakySchedules() *flakySchedules {
	schedules := &flakySchedules{ScheduleService: f.services.Schedules.(*mock.ScheduleService)}
	f.services.Schedules = schedules
	return schedules
}

// An unbind must take the app's schedules out of cron even when the job
// store fails, and a later rebind must not register a schedule twice.
func TestUnbindTakesSchedulesOutOfCronWhenJobLookupsFail(t *testing.T) {
	f := newFixture(t)
	jobs := &flakyJobs{JobService: f.services.Jobs.(*mock.JobService)}
	f.services.Jobs = jobs

	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)

	jobs.failing(true)
	rec := f.unbind("instance-1", "binding-1")
	jobs.failing(false)

	if rec.Code == nethttp.StatusOK {
		for name, schedule := range map[string]*core.Schedule{"nightly": f.nightly, "hourly": f.hourly, "owned": f.owned, "ping": f.ping} {
			if f.cron.Scheduled(schedule.GUID) {
				t.Errorf("%s: the unbind answered 200 but left the schedule in cron", name)
			}
		}
	} else if _, err := f.bindings.Get("binding-1"); err != nil {
		t.Fatalf("the unbind answered %d and must keep the binding for a retry, got %v", rec.Code, err)
	}

	expectStatus(t, "rebind", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusCreated)

	f.expectRunning("nightly", f.nightly)
	f.expectRunning("hourly", f.hourly)
	f.expectRunning("ping", f.ping)
	if n := f.cron.Count(); n != 3 {
		t.Errorf("cron holds %d entries after the rebind, want 3", n)
	}
}

// A failed pause answers 500 without the database's message, and the
// binding stays for Cloud Controller to retry.
func TestUnbindHidesAPauseFailure(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)

	schedules := f.flakySchedules()
	schedules.pauseErr = fmt.Errorf("%s", databaseError)

	for name, rec := range map[string]func() string{
		"unbind": func() string {
			rec := f.unbind("instance-1", "binding-1")
			expectStatus(t, "unbind", rec, nethttp.StatusInternalServerError)
			return rec.Body.String()
		},
		"deprovision": func() string {
			rec := f.deprovision("instance-1")
			expectStatus(t, "deprovision", rec, nethttp.StatusInternalServerError)
			return rec.Body.String()
		},
	} {
		if body := rec(); strings.Contains(body, "pq:") || strings.Contains(body, "10.0.0.5") {
			t.Errorf("%s: the description carries the database error: %s", name, body)
		}
	}

	if _, err := f.bindings.Get("binding-1"); err != nil {
		t.Errorf("the binding must stay for a retry, got %v", err)
	}
}

// Unbinding one of an app's two bindings must not pause the app, even when
// the store fails partway.
func TestUnbindingOneOfTwoBindingsNeverPausesTheApp(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)
	expectStatus(t, "bind again", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusCreated)

	schedules := f.flakySchedules()
	schedules.resumeErr = fmt.Errorf("%s", databaseError)

	rec := f.unbind("instance-1", "binding-1")
	if rec.Code != nethttp.StatusOK {
		// A retry after a failure would find the binding gone and answer
		// 410, so the app must already be running here.
		t.Errorf("unbind answered %d: %s", rec.Code, rec.Body.String())
	}

	f.expectRunning("nightly", f.nightly)
	f.expectRunning("ping", f.ping)
}

func TestBindRefusesTwoDifferentApps(t *testing.T) {
	f := newFixture(t)
	f.cf.PrepareApp("app-c", "space-a")
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)

	body := fmt.Sprintf(`{"service_id":%q,"plan_id":%q,"app_guid":"app-a","bind_resource":{"app_guid":"app-c"}}`, broker.ServiceID, broker.PlanID)
	rec := f.do(nethttp.MethodPut, "/v2/service_instances/instance-1/service_bindings/binding-1", body)
	expectStatus(t, "bind", rec, nethttp.StatusBadRequest)

	if f.bindings.Count() != 0 {
		t.Error("a bind naming two apps must not store a binding")
	}

	body = fmt.Sprintf(`{"service_id":%q,"plan_id":%q,"app_guid":"app-a","bind_resource":{"app_guid":"app-a"}}`, broker.ServiceID, broker.PlanID)
	expectStatus(t, "bind", f.do(nethttp.MethodPut, "/v2/service_instances/instance-1/service_bindings/binding-1", body), nethttp.StatusCreated)
}
