package broker_test

import (
	"context"
	"fmt"
	nethttp "net/http"
	"strings"
	"sync"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

// A rebind whose job lookups fail must not answer success while the job's
// schedules are enabled but missing from cron. They keep their mark, so the
// retry finds them and finishes the resume.
func TestRebindFinishesOnRetryWhenJobLookupsFail(t *testing.T) {
	f := newFixture(t)
	jobs := &flakyJobs{JobService: f.services.Jobs.(*mock.JobService)}
	f.services.Jobs = jobs

	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)
	expectStatus(t, "unbind", f.unbind("instance-1", "binding-1"), nethttp.StatusOK)

	jobs.failing(true)
	rec := f.bind("instance-1", "binding-2", "app-a")
	jobs.failing(false)

	expectStatus(t, "rebind", rec, nethttp.StatusInternalServerError)
	if body := rec.Body.String(); strings.Contains(body, "pq:") {
		t.Errorf("the description carries the database error: %s", body)
	}

	f.expectStopped("nightly", f.nightly, core.PausedByUnbind)
	f.expectStopped("hourly", f.hourly, core.PausedByUnbind)
	f.expectRunning("ping", f.ping)

	expectStatus(t, "retry", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusOK)

	f.expectRunning("nightly", f.nightly)
	f.expectRunning("hourly", f.hourly)
	f.expectRunning("ping", f.ping)
	if n := f.cron.Count(); n != 3 {
		t.Errorf("cron holds %d entries after the retry, want 3", n)
	}
}

// A resume whose store update fails leaves every schedule disabled, marked,
// and out of cron, so the retry starts over.
func TestRebindTakesSchedulesBackOutOfCronWhenTheResumeFails(t *testing.T) {
	f := newFixture(t)

	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)
	expectStatus(t, "unbind", f.unbind("instance-1", "binding-1"), nethttp.StatusOK)

	schedules := f.flakySchedules()
	schedules.resumeErr = fmt.Errorf("%s", databaseError)
	expectStatus(t, "rebind", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusInternalServerError)

	f.expectStopped("nightly", f.nightly, core.PausedByUnbind)
	f.expectStopped("ping", f.ping, core.PausedByUnbind)
	if n := f.cron.Count(); n != 0 {
		t.Errorf("cron holds %d entries after the failed resume, want 0", n)
	}

	schedules.resumeErr = nil
	expectStatus(t, "retry", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusOK)
	f.expectRunning("nightly", f.nightly)
	f.expectRunning("ping", f.ping)
}

// A repeated bind and a bind to an unknown instance are answered from the
// broker's own tables, so they don't need Cloud Controller. A new binding
// still does.
func TestBindAnswersFromItsOwnTablesFirst(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)

	f.cf.FailAppLookups(true)

	expectStatus(t, "repeated bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusOK)
	expectStatus(t, "unknown instance", f.bind("instance-9", "binding-9", "app-a"), nethttp.StatusNotFound)
	expectStatus(t, "clashing bind", f.bind("instance-1", "binding-1", "app-b"), nethttp.StatusConflict)
	expectStatus(t, "new binding", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusInternalServerError)

	if _, err := f.bindings.Get("binding-2"); err == nil {
		t.Error("a bind whose app lookup failed must not store a binding")
	}
}

// An unbind that cannot list the app's schedules keeps the binding for Cloud
// Controller to retry, and still takes out of cron what it paused.
func TestUnbindKeepsTheBindingWhenSchedulesCannotBeListed(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)

	schedules := f.services.Schedules.(*mock.ScheduleService)
	schedules.FailByApp(fmt.Errorf("%s", databaseError))

	rec := f.unbind("instance-1", "binding-1")
	expectStatus(t, "unbind", rec, nethttp.StatusInternalServerError)
	if body := rec.Body.String(); strings.Contains(body, "pq:") {
		t.Errorf("the description carries the database error: %s", body)
	}
	if _, err := f.bindings.Get("binding-1"); err != nil {
		t.Fatalf("the binding must stay for a retry, got %v", err)
	}

	f.expectStopped("nightly", f.nightly, core.PausedByUnbind)
	f.expectStopped("hourly", f.hourly, core.PausedByUnbind)
	f.expectStopped("ping", f.ping, core.PausedByUnbind)

	schedules.FailByApp(nil)
	expectStatus(t, "retry", f.unbind("instance-1", "binding-1"), nethttp.StatusOK)
	if n := f.cron.Count(); n != 0 {
		t.Errorf("cron holds %d entries after the unbind, want 0", n)
	}
}

// stuckCron fails every Delete, as a cron that cannot let go of an entry
// would.
type stuckCron struct {
	core.CronService
}

func (stuckCron) Delete(core.Runnable) error {
	return fmt.Errorf("cron could not remove the entry")
}

// An unbind that cannot take a schedule out of cron keeps the binding, so
// the schedule is never left firing for an app with no binding.
func TestUnbindKeepsTheBindingWhenCronCannotLetGo(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)

	f.services.Cron = stuckCron{f.cron}
	expectStatus(t, "unbind", f.unbind("instance-1", "binding-1"), nethttp.StatusInternalServerError)
	if _, err := f.bindings.Get("binding-1"); err != nil {
		t.Fatalf("the binding must stay for a retry, got %v", err)
	}

	f.services.Cron = f.cron
	expectStatus(t, "retry", f.unbind("instance-1", "binding-1"), nethttp.StatusOK)
	if n := f.cron.Count(); n != 0 {
		t.Errorf("cron holds %d entries after the unbind, want 0", n)
	}
}

// probedInfo records each Cloud Controller lookup that ran while the broker
// held the binding lock, which would stall every other broker operation and
// guarded route for as long as Cloud Controller took.
type probedInfo struct {
	core.InfoService
	lock *sync.RWMutex

	locker sync.Mutex
	locked []string
}

func (info *probedInfo) check(lookup string) {
	if info.lock.TryRLock() {
		info.lock.RUnlock()
		return
	}

	info.locker.Lock()
	defer info.locker.Unlock()

	info.locked = append(info.locked, lookup)
}

func (info *probedInfo) GetSpaceGUIDForApp(guid string) (string, error) {
	info.check("app " + guid)
	return info.InfoService.GetSpaceGUIDForApp(guid)
}

func (info *probedInfo) ServiceInstanceExists(ctx context.Context, guid string) (bool, error) {
	info.check("service instance " + guid)
	return info.InfoService.ServiceInstanceExists(ctx, guid)
}

func TestBrokerAsksCloudControllerWithoutTheLock(t *testing.T) {
	f := newFixture(t)
	info := &probedInfo{InfoService: f.services.Info, lock: &f.services.BindingLock}
	f.services.Info = info

	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)
	expectStatus(t, "replace the orphan", f.provision("instance-2", "space-a"), nethttp.StatusCreated)

	for _, lookup := range info.locked {
		t.Errorf("the broker asked Cloud Controller about %s while it held the binding lock", lookup)
	}
}
