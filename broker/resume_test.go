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

// recordingLog keeps every error line it is given, and passes every line on.
type recordingLog struct {
	core.LogService

	locker sync.Mutex
	errors []string
}

func (log *recordingLog) Error(tag, message string) {
	log.locker.Lock()
	log.errors = append(log.errors, message)
	log.locker.Unlock()

	log.LogService.Error(tag, message)
}

func (log *recordingLog) errorsNaming(text string) []string {
	log.locker.Lock()
	defer log.locker.Unlock()

	found := make([]string, 0)
	for _, line := range log.errors {
		if strings.Contains(line, text) {
			found = append(found, line)
		}
	}

	return found
}

// A schedule whose expression no longer parses can never go back in cron, so
// a rebind leaves it paused and marked, logs it, and resumes the others. A
// later bind resumes it once the expression parses again.
func TestRebindLeavesAnUnparseableScheduleBehind(t *testing.T) {
	f := newFixture(t)
	log := &recordingLog{LogService: f.services.Logger}
	f.services.Logger = log

	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)
	expectStatus(t, "unbind", f.unbind("instance-1", "binding-1"), nethttp.StatusOK)

	// A time zone that the zone data no longer holds.
	f.nightly.Expression = "CRON_TZ=Nowhere/Gone 0 2 * * *"

	expectStatus(t, "rebind", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusCreated)

	f.expectStopped("nightly", f.nightly, core.PausedByUnbind)
	f.expectRunning("hourly", f.hourly)
	f.expectRunning("ping", f.ping)
	f.expectStopped("owned", f.owned, "")

	logged := log.errorsNaming(f.nightly.GUID)
	if len(logged) == 0 {
		t.Errorf("no error names schedule %s", f.nightly.GUID)
	}
	for _, line := range logged {
		for _, secret := range []string{brokerPassword, callSecret, "auth_header"} {
			if strings.Contains(line, secret) {
				t.Errorf("the log carries %q: %s", secret, line)
			}
		}
	}

	expectStatus(t, "repeated bind", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusOK)
	f.expectStopped("nightly", f.nightly, core.PausedByUnbind)
	if n := f.cron.Count(); n != 2 {
		t.Errorf("cron holds %d entries, want 2", n)
	}

	f.nightly.Expression = "0 2 * * *"
	expectStatus(t, "bind after the fix", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusOK)
	f.expectRunning("nightly", f.nightly)
	if n := f.cron.Count(); n != 3 {
		t.Errorf("cron holds %d entries, want 3", n)
	}
}

// withToken replaces the broker's basic auth with a scheduler API token.
func withToken(token string) option {
	return func(req *nethttp.Request) { req.Header.Set("Authorization", token) }
}

// The owner of a schedule that a rebind left paused can delete it and create
// it again, and can delete a job or call that owns a schedule with no cron
// entry.
func TestOwnerCanReplaceAScheduleTheRebindLeftPaused(t *testing.T) {
	f := newFixture(t)
	f.services.RequireBinding = true
	f.services.Auth.(*mock.AuthService).Developer("jeremy", "space-a")
	token := withToken("jeremy")

	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)
	expectStatus(t, "unbind", f.unbind("instance-1", "binding-1"), nethttp.StatusOK)

	f.nightly.Expression = "CRON_TZ=Nowhere/Gone 0 2 * * *"
	expectStatus(t, "rebind", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusCreated)
	f.expectStopped("nightly", f.nightly, core.PausedByUnbind)

	jobSchedules := "/jobs/" + f.job.GUID + "/schedules/"
	expectStatus(t, "delete the paused schedule",
		f.do(nethttp.MethodDelete, jobSchedules+f.nightly.GUID, "", token), nethttp.StatusNoContent)
	expectStatus(t, "delete the owner-disabled schedule",
		f.do(nethttp.MethodDelete, jobSchedules+f.owned.GUID, "", token), nethttp.StatusNoContent)
	for name, schedule := range map[string]*core.Schedule{"nightly": f.nightly, "owned": f.owned} {
		if _, err := f.services.Schedules.Get(schedule.GUID); err == nil {
			t.Errorf("%s is still stored after its delete", name)
		}
	}

	rec := f.do(nethttp.MethodPost, "/jobs/"+f.job.GUID+"/schedules",
		`{"enabled": true, "expression": "0 2 * * *", "expression_type": "cron_expression"}`, token)
	expectStatus(t, "recreate", rec, nethttp.StatusCreated)
	guid, _ := decode(t, rec)["guid"].(string)
	recreated, err := f.services.Schedules.Get(guid)
	if err != nil {
		t.Fatalf("the recreated schedule %q is not stored: %v", guid, err)
	}
	f.expectRunning("recreated", recreated)

	// A call whose only schedule has no cron entry can be deleted too.
	if err := f.cron.Delete(core.NewCallRun(f.call, f.ping, f.services)); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, "delete the call",
		f.do(nethttp.MethodDelete, "/calls/"+f.call.GUID, "", token), nethttp.StatusNoContent)

	// So can a job with one schedule in cron and one out of it.
	if err := f.cron.Delete(core.NewJobRun(f.job, f.hourly, f.services)); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, "delete the job",
		f.do(nethttp.MethodDelete, "/jobs/"+f.job.GUID, "", token), nethttp.StatusNoContent)

	if n := f.cron.Count(); n != 0 {
		t.Errorf("cron holds %d entries after every owner was deleted, want 0", n)
	}
}

// racingSchedules runs race once, inside the first Delete, either before or
// after the row goes.
type racingSchedules struct {
	*mock.ScheduleService
	race  func()
	after bool

	locker sync.Mutex
	raced  bool
}

func (schedules *racingSchedules) Delete(schedule *core.Schedule) error {
	schedules.locker.Lock()
	raced := schedules.raced
	schedules.raced = true
	schedules.locker.Unlock()

	if raced {
		return schedules.ScheduleService.Delete(schedule)
	}

	if !schedules.after {
		schedules.race()
	}
	err := schedules.ScheduleService.Delete(schedule)
	if schedules.after {
		schedules.race()
	}

	return err
}

// A rebind that runs while the owner deletes a paused schedule must not
// leave a cron entry behind for the deleted schedule, whichever side of the
// row delete it lands on.
func TestDeletingAPausedScheduleDuringARebind(t *testing.T) {
	for name, after := range map[string]bool{"before the row goes": false, "after the row goes": true} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.services.Auth.(*mock.AuthService).Developer("jeremy", "space-a")

			expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
			expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)
			expectStatus(t, "unbind", f.unbind("instance-1", "binding-1"), nethttp.StatusOK)
			f.expectStopped("nightly", f.nightly, core.PausedByUnbind)

			f.services.Schedules = &racingSchedules{
				ScheduleService: f.services.Schedules.(*mock.ScheduleService),
				after:           after,
				race: func() {
					expectStatus(t, "rebind", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusCreated)
				},
			}

			expectStatus(t, "delete the paused schedule",
				f.do(nethttp.MethodDelete, "/jobs/"+f.job.GUID+"/schedules/"+f.nightly.GUID, "", withToken("jeremy")),
				nethttp.StatusNoContent)

			if _, err := f.services.Schedules.Get(f.nightly.GUID); err == nil {
				t.Error("the deleted schedule is still stored")
			}
			if f.cron.Scheduled(f.nightly.GUID) {
				t.Error("the deleted schedule is still in cron")
			}
			f.expectRunning("hourly", f.hourly)
			f.expectRunning("ping", f.ping)
			if n := f.cron.Count(); n != 2 {
				t.Errorf("cron holds %d entries, want 2", n)
			}
		})
	}
}

// failingDeletes refuses every row delete, as a lost database would.
type failingDeletes struct {
	*mock.ScheduleService
}

func (failingDeletes) Delete(*core.Schedule) error {
	return fmt.Errorf("%s", databaseError)
}

// A delete whose row delete fails leaves the schedule stored and in cron.
func TestDeletingAScheduleKeepsItsCronEntryWhenTheStoreFails(t *testing.T) {
	f := newFixture(t)
	f.services.Auth.(*mock.AuthService).Developer("jeremy", "space-a")
	f.services.Schedules = failingDeletes{f.services.Schedules.(*mock.ScheduleService)}

	expectStatus(t, "delete",
		f.do(nethttp.MethodDelete, "/jobs/"+f.job.GUID+"/schedules/"+f.nightly.GUID, "", withToken("jeremy")),
		nethttp.StatusInternalServerError)

	if _, err := f.services.Schedules.Get(f.nightly.GUID); err != nil {
		t.Errorf("the schedule must stay stored, got %v", err)
	}
	if !f.cron.Scheduled(f.nightly.GUID) {
		t.Error("the schedule must stay in cron")
	}
}

// fullCron refuses every Add for a reason other than the expression.
type fullCron struct {
	core.CronService
}

func (fullCron) Add(core.Runnable) error {
	return fmt.Errorf("cron is full")
}

// A cron failure that has nothing to do with the expression still fails the
// rebind, so Cloud Controller retries and the retry resumes the schedules.
func TestRebindFailsWhenCronRefusesAValidSchedule(t *testing.T) {
	f := newFixture(t)

	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)
	expectStatus(t, "unbind", f.unbind("instance-1", "binding-1"), nethttp.StatusOK)

	f.services.Cron = fullCron{f.cron}
	expectStatus(t, "rebind", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusInternalServerError)
	f.expectStopped("nightly", f.nightly, core.PausedByUnbind)
	f.expectStopped("ping", f.ping, core.PausedByUnbind)

	f.services.Cron = f.cron
	expectStatus(t, "retry", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusOK)
	f.expectRunning("nightly", f.nightly)
	f.expectRunning("ping", f.ping)
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
