package routes

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

// enforce turns require_binding on with an empty binding store.
func (tn *tenancy) enforce() *mock.BindingService {
	bindings := mock.NewBindingService()
	tn.services.Bindings = bindings
	tn.services.RequireBinding = true
	return bindings
}

// gatedOnB is every route that require_binding guards, aimed at tenant B.
func (tn *tenancy) gatedOnB() []request {
	job, call := "/jobs/"+tn.jobB.GUID, "/calls/"+tn.callB.GUID
	schedule := `{"enabled": true, "expression": "0 3 * * *", "expression_type": "cron_expression"}`
	return []request{
		{http.MethodPost, "/jobs?app_guid=app-b", `{"name": "nightly", "command": "env"}`, http.StatusCreated, scoped},
		{http.MethodPost, "/calls?app_guid=app-b", `{"name": "hook", "url": "https://example.com", "auth_header": "x"}`, http.StatusCreated, scoped},
		{http.MethodPost, job + "/schedules", schedule, http.StatusCreated, object},
		{http.MethodPost, call + "/schedules", schedule, http.StatusCreated, object},
		{http.MethodPost, job + "/execute", `{}`, http.StatusCreated, object},
		{http.MethodPost, call + "/execute", `{}`, http.StatusCreated, object},
	}
}

func TestRequireBindingRefusesAnUnboundApp(t *testing.T) {
	tn := newTenancy(t)
	tn.enforce()

	for _, r := range tn.gatedOnB() {
		rec := tn.do(r.method, r.path, r.body, tn.userB)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s: status %d, want 403, body %s", r.method, r.path, rec.Code, rec.Body)
		}
		if !strings.Contains(rec.Body.String(), "binding_required") || !strings.Contains(rec.Body.String(), "cf bind-service") {
			t.Errorf("%s %s: want a binding_required error that names cf bind-service, got %s", r.method, r.path, rec.Body)
		}
	}

	tn.assertBUntouched(t)
}

func TestRequireBindingAllowsABoundApp(t *testing.T) {
	tn := newTenancy(t)
	bindings := tn.enforce()
	if _, err := bindings.Persist(&core.ServiceBinding{BindingID: "binding-b", InstanceID: "instance-b", AppGUID: "app-b"}); err != nil {
		t.Fatal(err)
	}

	tn.check(t, tn.userB, tn.gatedOnB(), allowed)
	if n := tn.runner.count(); n != 2 {
		t.Errorf("%d runs were dispatched, want 2", n)
	}
}

// Another app's binding does not open tenant B's app.
func TestRequireBindingChecksTheNamedApp(t *testing.T) {
	tn := newTenancy(t)
	bindings := tn.enforce()
	if _, err := bindings.Persist(&core.ServiceBinding{BindingID: "binding-a", InstanceID: "instance-a", AppGUID: "app-a"}); err != nil {
		t.Fatal(err)
	}

	tn.check(t, tn.userB, tn.gatedOnB(), denied(http.StatusForbidden, http.StatusForbidden, http.StatusForbidden))
	tn.assertBUntouched(t)
}

func TestRequireBindingLetsAnAdminThrough(t *testing.T) {
	tn := newTenancy(t)
	tn.enforce()

	tn.check(t, tn.admin, tn.gatedOnB(), allowed)
}

func TestRequireBindingOffAllowsAnUnboundApp(t *testing.T) {
	tn := newTenancy(t)
	tn.services.Bindings = mock.NewBindingService()

	tn.check(t, tn.userB, tn.gatedOnB(), allowed)
}

// The gate runs after the space check, so another tenant still gets the
// answer it got before and learns nothing about the binding.
func TestRequireBindingKeepsTheSpaceCheckFirst(t *testing.T) {
	tn := newTenancy(t)
	tn.enforce()

	tn.check(t, tn.userA, tn.gatedOnB(), denied(http.StatusForbidden, http.StatusNotFound, http.StatusNotFound))
	for _, auth := range []string{"", "Bearer not-a-token"} {
		tn.check(t, auth, tn.gatedOnB(), denied(http.StatusUnauthorized, http.StatusUnauthorized, http.StatusUnauthorized))
	}
	tn.assertBUntouched(t)
}

// Reading, listing, and deleting stay open to an unbound app, so its owner
// can still see and clean up what it has.
func TestRequireBindingLeavesOtherRoutesOpen(t *testing.T) {
	tn := newTenancy(t)
	tn.enforce()

	job, call := "/jobs/"+tn.jobB.GUID, "/calls/"+tn.callB.GUID
	tn.check(t, tn.userB, []request{
		{http.MethodGet, "/jobs?space_guid=space-b", "", http.StatusOK, scoped},
		{http.MethodGet, job, "", http.StatusOK, object},
		{http.MethodGet, call + "/schedules", "", http.StatusOK, object},
		{http.MethodGet, job + "/history", "", http.StatusOK, object},
		{http.MethodDelete, job + "/schedules/" + tn.jobSchedB.GUID, "", http.StatusNoContent, object},
		{http.MethodDelete, call, "", http.StatusNoContent, object},
	}, allowed)
}

func TestRequireBindingFailsClosed(t *testing.T) {
	t.Run("when the binding store fails", func(t *testing.T) {
		tn := newTenancy(t)
		tn.enforce().Fail(fmt.Errorf("pq: connection refused"))

		tn.check(t, tn.userB, tn.gatedOnB(), denied(http.StatusInternalServerError, http.StatusInternalServerError, http.StatusInternalServerError))
		tn.assertBUntouched(t)
	})

	t.Run("when the broker is not running", func(t *testing.T) {
		tn := newTenancy(t)
		tn.services.RequireBinding = true

		tn.check(t, tn.userB, tn.gatedOnB(), denied(http.StatusInternalServerError, http.StatusInternalServerError, http.StatusInternalServerError))
		tn.assertBUntouched(t)
	})
}

// A guarded route waits while the broker changes bindings, so it sees the
// unbind that the broker was making instead of slipping in before it.
func TestRequireBindingWaitsForTheBroker(t *testing.T) {
	tn := newTenancy(t)
	bindings := tn.enforce()
	if _, err := bindings.Persist(&core.ServiceBinding{BindingID: "binding-b", InstanceID: "instance-b", AppGUID: "app-b"}); err != nil {
		t.Fatal(err)
	}

	// Stand in for the broker in the middle of an unbind.
	tn.services.BindingLock.Lock()

	done := make(chan *httptest.ResponseRecorder)
	go func() {
		done <- tn.do(http.MethodPost, "/jobs/"+tn.jobB.GUID+"/schedules",
			`{"enabled": true, "expression": "0 3 * * *", "expression_type": "cron_expression"}`, tn.userB)
	}()

	select {
	case rec := <-done:
		tn.services.BindingLock.Unlock()
		t.Fatalf("the route answered %d while the broker held the lock", rec.Code)
	case <-time.After(100 * time.Millisecond):
	}

	if err := bindings.Delete("binding-b"); err != nil {
		t.Fatal(err)
	}
	tn.services.BindingLock.Unlock()

	if rec := <-done; rec.Code != http.StatusForbidden {
		t.Errorf("status %d after the unbind, want 403, body %s", rec.Code, rec.Body)
	}
	tn.assertBUntouched(t)
}

// stallingBody holds its first read until release is closed, as a client
// that stops sending its body would. started is closed when the route first
// reads it.
type stallingBody struct {
	body     io.Reader
	started  chan struct{}
	release  chan struct{}
	stalling sync.Once
}

func newStallingBody(body string) *stallingBody {
	return &stallingBody{
		body:    strings.NewReader(body),
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (body *stallingBody) Read(p []byte) (int, error) {
	body.stalling.Do(func() {
		close(body.started)
		<-body.release
	})

	return body.body.Read(p)
}

// A client that stalls its body must not hold the binding lock, or the
// broker and every other guarded route would wait on that one client.
func TestRequireBindingNeverHoldsTheLockWhileTheClientSends(t *testing.T) {
	tn := newTenancy(t)
	bindings := tn.enforce()
	if _, err := bindings.Persist(&core.ServiceBinding{BindingID: "binding-b", InstanceID: "instance-b", AppGUID: "app-b"}); err != nil {
		t.Fatal(err)
	}

	for _, r := range tn.gatedOnB() {
		body := newStallingBody(r.body)
		done := make(chan *httptest.ResponseRecorder)
		go func() {
			done <- tn.serve(r.method, r.path, body, tn.userB)
		}()

		select {
		case <-body.started:
		case rec := <-done:
			t.Fatalf("%s %s: answered %d without reading the body", r.method, r.path, rec.Code)
		case <-time.After(5 * time.Second):
			t.Fatalf("%s %s: never read the body", r.method, r.path)
		}

		if tn.services.BindingLock.TryLock() {
			tn.services.BindingLock.Unlock()
		} else {
			t.Errorf("%s %s: the route holds the binding lock while the client sends its body", r.method, r.path)
		}

		close(body.release)
		if rec := <-done; rec.Code != r.ok {
			t.Errorf("%s %s: status %d, want %d, body %s", r.method, r.path, rec.Code, r.ok, rec.Body)
		}
	}
}

// lockProbe records each store step that ran while the binding lock was
// free, which would let the broker unbind the app between the route's
// binding check and its write.
type lockProbe struct {
	lock *sync.RWMutex

	locker   sync.Mutex
	unlocked []string
}

func (probe *lockProbe) check(step string) {
	if !probe.lock.TryLock() {
		return
	}
	probe.lock.Unlock()

	probe.locker.Lock()
	defer probe.locker.Unlock()

	probe.unlocked = append(probe.unlocked, step)
}

type probedJobs struct {
	core.JobService
	probe *lockProbe
}

func (jobs probedJobs) Persist(job *core.Job) (*core.Job, error) {
	jobs.probe.check("persist job")
	return jobs.JobService.Persist(job)
}

type probedCalls struct {
	core.CallService
	probe *lockProbe
}

func (calls probedCalls) Persist(call *core.Call) (*core.Call, error) {
	calls.probe.check("persist call")
	return calls.CallService.Persist(call)
}

type probedSchedules struct {
	core.ScheduleService
	probe *lockProbe
}

func (schedules probedSchedules) Persist(schedule *core.Schedule) (*core.Schedule, error) {
	schedules.probe.check("persist schedule")
	return schedules.ScheduleService.Persist(schedule)
}

type probedExecutions struct {
	core.ExecutionService
	probe *lockProbe
}

func (executions probedExecutions) Persist(execution *core.Execution) (*core.Execution, error) {
	executions.probe.check("persist execution")
	return executions.ExecutionService.Persist(execution)
}

type probedCron struct {
	core.CronService
	probe *lockProbe
}

func (cron probedCron) Add(runnable core.Runnable) error {
	cron.probe.check("add to cron")
	return cron.CronService.Add(runnable)
}

type probedRunner struct {
	core.RunService
	probe *lockProbe
}

func (runner probedRunner) Execute(services *core.Services, execution *core.Execution, executable core.Executable) {
	runner.probe.check("dispatch run")
	runner.RunService.Execute(services, execution, executable)
}

// Each guarded route stores its work, puts it in cron, and dispatches it
// while it holds the binding lock, so an unbind cannot land between the
// route's last binding check and its write.
func TestRequireBindingHoldsTheLockWhileItStores(t *testing.T) {
	tn := newTenancy(t)
	bindings := tn.enforce()
	if _, err := bindings.Persist(&core.ServiceBinding{BindingID: "binding-b", InstanceID: "instance-b", AppGUID: "app-b"}); err != nil {
		t.Fatal(err)
	}

	probe := &lockProbe{lock: &tn.services.BindingLock}
	tn.services.Jobs = probedJobs{tn.services.Jobs, probe}
	tn.services.Calls = probedCalls{tn.services.Calls, probe}
	tn.services.Schedules = probedSchedules{tn.services.Schedules, probe}
	tn.services.Executions = probedExecutions{tn.services.Executions, probe}
	tn.services.Cron = probedCron{tn.services.Cron, probe}
	tn.services.Runner = probedRunner{tn.services.Runner, probe}

	tn.check(t, tn.userB, tn.gatedOnB(), allowed)

	for _, step := range probe.unlocked {
		t.Errorf("%s ran without the binding lock", step)
	}
	if n := tn.runner.count(); n != 2 {
		t.Errorf("%d runs were dispatched, want 2", n)
	}
}
