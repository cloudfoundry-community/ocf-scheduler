package broker_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/broker"
	"github.com/cloudfoundry-community/ocf-scheduler/cf"
	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/cron"
	schedhttp "github.com/cloudfoundry-community/ocf-scheduler/http"
	"github.com/cloudfoundry-community/ocf-scheduler/logger"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

const (
	brokerUser     = "scheduler-broker"
	brokerPassword = "broker-password-do-not-leak"
	apiURL         = "https://scheduler.system.example.com"
	callSecret     = "Basic Y2FsbC1zZWNyZXQtZG8tbm90LWxlYWs="
	apiVersion     = "2.17"
)

// fixture serves the scheduler with the broker mounted, as main does, on
// mocks. App app-a lives in space-a and has a job with three schedules, one
// of them disabled by its owner, and a call with one schedule. App app-b
// lives in space-b and has nothing.
type fixture struct {
	t        *testing.T
	services *core.Services
	cf       *mock.CFClient
	cron     *cron.CronService
	server   nethttp.Handler

	instances *mock.InstanceService
	bindings  *mock.BindingService

	job                    *core.Job
	call                   *core.Call
	nightly, hourly, owned *core.Schedule
	ping                   *core.Schedule
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	client, err := mock.NewCFClient()
	if err != nil {
		t.Fatal(err)
	}
	client.PrepareApp("app-a", "space-a")
	client.PrepareApp("app-b", "space-b")

	log := logger.New()
	jobs := mock.NewJobService()
	calls := mock.NewCallService()
	schedules := mock.NewScheduleService()
	schedules.Owners(jobs, calls)

	f := &fixture{
		t:         t,
		cf:        client,
		cron:      cron.NewCronService(log, cron.Rules{}),
		instances: mock.NewInstanceService(),
		bindings:  mock.NewBindingService(),
	}

	f.services = &core.Services{
		Info:       cf.NewInfoService(client),
		Jobs:       jobs,
		Calls:      calls,
		Schedules:  schedules,
		Executions: mock.NewExecutionService(),
		Cron:       f.cron,
		Logger:     log,
		Auth:       mock.NewAuthService(),
		Instances:  f.instances,
		Bindings:   f.bindings,
	}

	handler, err := broker.NewHandler(f.services, broker.Config{
		Enabled:  true,
		Username: brokerUser,
		Password: brokerPassword,
		APIURL:   apiURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	f.server = schedhttp.Server("127.0.0.1:0", f.services, handler).Handler

	f.job, err = jobs.Persist(&core.Job{Name: "backup", Command: "true", AppGUID: "app-a", SpaceGUID: "space-a"})
	if err != nil {
		t.Fatal(err)
	}
	f.call, err = calls.Persist(&core.Call{Name: "ping", URL: "https://example.com", AuthHeader: callSecret, AppGUID: "app-a", SpaceGUID: "space-a"})
	if err != nil {
		t.Fatal(err)
	}

	f.nightly = f.schedule(f.job, true)
	f.hourly = f.schedule(f.job, true)
	f.owned = f.schedule(f.job, false)
	f.ping = f.schedule(f.call, true)

	return f
}

// schedule stores a schedule and puts it in cron, as the create routes do
// even for a disabled one.
func (f *fixture) schedule(executable core.Executable, enabled bool) *core.Schedule {
	f.t.Helper()

	schedule := &core.Schedule{Enabled: enabled, Expression: "0 2 * * *", ExpressionType: "cron_expression", RefType: executable.Type()}
	var run core.Runnable
	if job, err := executable.ToJob(); err == nil {
		schedule.RefGUID = job.GUID
		run = core.NewJobRun(job, schedule, f.services)
	} else {
		call, _ := executable.ToCall()
		schedule.RefGUID = call.GUID
		run = core.NewCallRun(call, schedule, f.services)
	}

	if _, err := f.services.Schedules.Persist(schedule); err != nil {
		f.t.Fatal(err)
	}
	if err := f.cron.Add(run); err != nil {
		f.t.Fatal(err)
	}

	return schedule
}

type option func(*nethttp.Request)

func withoutVersion() option {
	return func(req *nethttp.Request) { req.Header.Del("X-Broker-API-Version") }
}

func withVersion(version string) option {
	return func(req *nethttp.Request) { req.Header.Set("X-Broker-API-Version", version) }
}

func withAuth(user, password string) option {
	return func(req *nethttp.Request) { req.SetBasicAuth(user, password) }
}

func withoutAuth() option {
	return func(req *nethttp.Request) { req.Header.Del("Authorization") }
}

func withIdentity(user string) option {
	return func(req *nethttp.Request) {
		value := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`{"user_id":%q}`, user)))
		req.Header.Set("X-Broker-API-Originating-Identity", "cloudfoundry "+value)
	}
}

// do sends a broker request with the broker's credentials and API version
// unless an option changes them. Every answer is checked for the broker
// password and the call's auth header.
func (f *fixture) do(method, path, body string, options ...option) *httptest.ResponseRecorder {
	f.t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Broker-API-Version", apiVersion)
	req.SetBasicAuth(brokerUser, brokerPassword)
	for _, apply := range options {
		apply(req)
	}

	rec := httptest.NewRecorder()
	f.server.ServeHTTP(rec, req)

	for _, secret := range []string{brokerPassword, callSecret, "auth_header"} {
		if strings.Contains(rec.Body.String(), secret) {
			f.t.Fatalf("%s %s leaked %q: %s", method, path, secret, rec.Body.String())
		}
	}

	return rec
}

func (f *fixture) provision(instanceID, spaceGUID string, options ...option) *httptest.ResponseRecorder {
	f.t.Helper()

	body := fmt.Sprintf(`{"service_id":%q,"plan_id":%q,"organization_guid":"org-1","space_guid":%q,"context":{"platform":"cloudfoundry","space_guid":%q}}`,
		broker.ServiceID, broker.PlanID, spaceGUID, spaceGUID)

	return f.do(nethttp.MethodPut, "/v2/service_instances/"+instanceID, body, options...)
}

func (f *fixture) bind(instanceID, bindingID, appGUID string) *httptest.ResponseRecorder {
	f.t.Helper()

	body := fmt.Sprintf(`{"service_id":%q,"plan_id":%q,"bind_resource":{"app_guid":%q}}`, broker.ServiceID, broker.PlanID, appGUID)
	if len(appGUID) == 0 {
		body = fmt.Sprintf(`{"service_id":%q,"plan_id":%q}`, broker.ServiceID, broker.PlanID)
	}

	return f.do(nethttp.MethodPut, "/v2/service_instances/"+instanceID+"/service_bindings/"+bindingID, body)
}

func (f *fixture) unbind(instanceID, bindingID string) *httptest.ResponseRecorder {
	f.t.Helper()

	return f.do(nethttp.MethodDelete, fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s?service_id=%s&plan_id=%s",
		instanceID, bindingID, broker.ServiceID, broker.PlanID), "")
}

func (f *fixture) deprovision(instanceID string) *httptest.ResponseRecorder {
	f.t.Helper()

	return f.do(nethttp.MethodDelete, fmt.Sprintf("/v2/service_instances/%s?service_id=%s&plan_id=%s",
		instanceID, broker.ServiceID, broker.PlanID), "")
}

// state reads a schedule back from the store.
func (f *fixture) state(schedule *core.Schedule) *core.Schedule {
	f.t.Helper()

	stored, err := f.services.Schedules.Get(schedule.GUID)
	if err != nil {
		f.t.Fatal(err)
	}

	return stored
}

// expectRunning checks that a schedule is enabled, unmarked, and in cron.
func (f *fixture) expectRunning(name string, schedule *core.Schedule) {
	f.t.Helper()

	stored := f.state(schedule)
	if !stored.Enabled || stored.PausedBy != "" || !f.cron.Scheduled(schedule.GUID) {
		f.t.Errorf("%s: want enabled, unmarked, and in cron; got enabled=%t paused_by=%q in_cron=%t",
			name, stored.Enabled, stored.PausedBy, f.cron.Scheduled(schedule.GUID))
	}
}

// expectStopped checks that a schedule is disabled with the given mark and
// out of cron.
func (f *fixture) expectStopped(name string, schedule *core.Schedule, pausedBy string) {
	f.t.Helper()

	stored := f.state(schedule)
	if stored.Enabled || stored.PausedBy != pausedBy || f.cron.Scheduled(schedule.GUID) {
		f.t.Errorf("%s: want disabled with paused_by=%q and out of cron; got enabled=%t paused_by=%q in_cron=%t",
			name, pausedBy, stored.Enabled, stored.PausedBy, f.cron.Scheduled(schedule.GUID))
	}
}

func expectStatus(t *testing.T, what string, rec *httptest.ResponseRecorder, want int) {
	t.Helper()

	if rec.Code != want {
		t.Fatalf("%s: got %d, want %d: %s", what, rec.Code, want, rec.Body.String())
	}
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not decode %q: %v", rec.Body.String(), err)
	}

	return body
}

func TestCatalog(t *testing.T) {
	f := newFixture(t)

	rec := f.do(nethttp.MethodGet, "/v2/catalog", "")
	expectStatus(t, "catalog", rec, nethttp.StatusOK)

	var catalog struct {
		Services []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Bindable bool   `json:"bindable"`
			Plans    []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"plans"`
		} `json:"services"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}

	if len(catalog.Services) != 1 || len(catalog.Services[0].Plans) != 1 {
		t.Fatalf("want one service with one plan, got %s", rec.Body.String())
	}

	service := catalog.Services[0]
	if service.ID != broker.ServiceID || service.Name != broker.ServiceName || !service.Bindable {
		t.Errorf("unexpected service %+v", service)
	}
	if service.Plans[0].ID != broker.PlanID || service.Plans[0].Name != broker.PlanName {
		t.Errorf("unexpected plan %+v", service.Plans[0])
	}
}

func TestBrokerAPIVersionIsRequired(t *testing.T) {
	f := newFixture(t)

	for name, options := range map[string][]option{
		"missing":                     {withoutVersion()},
		"too old":                     {withVersion("2.12")},
		"missing with no auth":        {withoutVersion(), withoutAuth()},
		"too old with a bad password": {withVersion("2.11"), withAuth(brokerUser, "wrong")},
	} {
		t.Run(name, func(t *testing.T) {
			rec := f.do(nethttp.MethodGet, "/v2/catalog", "", options...)
			expectStatus(t, name, rec, nethttp.StatusPreconditionFailed)
		})
	}

	if f.instances.Count() != 0 {
		t.Fatal("nothing should have been provisioned")
	}
	rec := f.provision("instance-1", "space-a", withoutVersion())
	expectStatus(t, "provision without a version", rec, nethttp.StatusPreconditionFailed)
	if f.instances.Count() != 0 {
		t.Error("a request without a version must not provision")
	}
}

func TestBrokerBasicAuth(t *testing.T) {
	f := newFixture(t)

	for name, options := range map[string][]option{
		"no credentials":   {withoutAuth()},
		"wrong password":   {withAuth(brokerUser, "wrong")},
		"wrong username":   {withAuth("admin", brokerPassword)},
		"empty password":   {withAuth(brokerUser, "")},
		"a bearer token":   {func(req *nethttp.Request) { req.Header.Set("Authorization", "Bearer jeremy") }},
		"swapped user/pwd": {withAuth(brokerPassword, brokerUser)},
	} {
		t.Run(name, func(t *testing.T) {
			rec := f.do(nethttp.MethodGet, "/v2/catalog", "", options...)
			expectStatus(t, name, rec, nethttp.StatusUnauthorized)

			rec = f.provision("instance-1", "space-a", options...)
			expectStatus(t, name+" provision", rec, nethttp.StatusUnauthorized)
		})
	}

	if f.instances.Count() != 0 {
		t.Error("an unauthenticated request must not provision")
	}
}

// The broker credentials open only /v2. The scheduler API still wants a
// Cloud Foundry token.
func TestBrokerCredentialsDoNotOpenTheSchedulerAPI(t *testing.T) {
	f := newFixture(t)

	rec := f.do(nethttp.MethodGet, "/jobs?space_guid=space-a", "")
	if rec.Code != nethttp.StatusUnauthorized {
		t.Fatalf("GET /jobs with broker credentials: got %d, want 401: %s", rec.Code, rec.Body.String())
	}
}

func TestProvision(t *testing.T) {
	f := newFixture(t)

	rec := f.provision("instance-1", "space-a", withIdentity("user-1"))
	expectStatus(t, "provision", rec, nethttp.StatusCreated)

	instance, err := f.instances.Get("instance-1")
	if err != nil {
		t.Fatal(err)
	}
	if instance.SpaceGUID != "space-a" || instance.OrgGUID != "org-1" || instance.Context["platform"] != "cloudfoundry" {
		t.Errorf("unexpected instance %+v", instance)
	}

	t.Run("repeating it identically answers 200", func(t *testing.T) {
		expectStatus(t, "repeat", f.provision("instance-1", "space-a"), nethttp.StatusOK)
	})

	t.Run("the same ID with other attributes answers 409", func(t *testing.T) {
		expectStatus(t, "conflict", f.provision("instance-1", "space-b"), nethttp.StatusConflict)
	})

	t.Run("fetching it returns the service and plan", func(t *testing.T) {
		rec := f.do(nethttp.MethodGet, "/v2/service_instances/instance-1", "")
		expectStatus(t, "fetch", rec, nethttp.StatusOK)

		body := decode(t, rec)
		if body["service_id"] != broker.ServiceID || body["plan_id"] != broker.PlanID {
			t.Errorf("unexpected instance %v", body)
		}
	})

	t.Run("fetching an unknown instance answers 404", func(t *testing.T) {
		expectStatus(t, "fetch", f.do(nethttp.MethodGet, "/v2/service_instances/nope", ""), nethttp.StatusNotFound)
	})

	if f.instances.Count() != 1 {
		t.Errorf("want 1 instance, got %d", f.instances.Count())
	}
}

func TestProvisionTakesTheSpaceFromTheContext(t *testing.T) {
	f := newFixture(t)

	body := fmt.Sprintf(`{"service_id":%q,"plan_id":%q,"context":{"platform":"cloudfoundry","organization_guid":"org-9","space_guid":"space-b"}}`, broker.ServiceID, broker.PlanID)
	expectStatus(t, "provision", f.do(nethttp.MethodPut, "/v2/service_instances/instance-1", body), nethttp.StatusCreated)

	instance, err := f.instances.Get("instance-1")
	if err != nil {
		t.Fatal(err)
	}
	if instance.SpaceGUID != "space-b" || instance.OrgGUID != "org-9" {
		t.Errorf("unexpected instance %+v", instance)
	}
}

func TestProvisionRejectsBadRequests(t *testing.T) {
	f := newFixture(t)

	for name, body := range map[string]string{
		"no space":        fmt.Sprintf(`{"service_id":%q,"plan_id":%q}`, broker.ServiceID, broker.PlanID),
		"unknown service": fmt.Sprintf(`{"service_id":"other","plan_id":%q,"space_guid":"space-a"}`, broker.PlanID),
		"unknown plan":    fmt.Sprintf(`{"service_id":%q,"plan_id":"other","space_guid":"space-a"}`, broker.ServiceID),
	} {
		t.Run(name, func(t *testing.T) {
			expectStatus(t, name, f.do(nethttp.MethodPut, "/v2/service_instances/instance-1", body), nethttp.StatusBadRequest)
		})
	}

	if f.instances.Count() != 0 {
		t.Error("a bad request must not provision")
	}
}

func TestProvisionAllowsOneInstancePerSpace(t *testing.T) {
	f := newFixture(t)

	expectStatus(t, "first", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	f.cf.PrepareServiceInstance("instance-1")

	rec := f.provision("instance-2", "space-a")
	expectStatus(t, "second in the same space", rec, nethttp.StatusConflict)
	if !strings.Contains(rec.Body.String(), "only one") {
		t.Errorf("want a description that names the one-per-space rule, got %s", rec.Body.String())
	}

	expectStatus(t, "another space", f.provision("instance-3", "space-b"), nethttp.StatusCreated)

	if _, err := f.instances.Get("instance-2"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("instance-2 should not exist, got %v", err)
	}
}

// After cf purge-service-instance, Cloud Controller no longer knows the
// instance but the broker still holds it. A new instance replaces it, and
// its app's schedules pause as they would on deprovision.
func TestProvisionReplacesAnOrphanedInstance(t *testing.T) {
	f := newFixture(t)

	expectStatus(t, "first", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)

	expectStatus(t, "replacement", f.provision("instance-2", "space-a"), nethttp.StatusCreated)

	if _, err := f.instances.Get("instance-1"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("the orphan should be gone, got %v", err)
	}
	if f.bindings.Count() != 0 {
		t.Errorf("the orphan's binding should be gone, got %d bindings", f.bindings.Count())
	}

	f.expectStopped("nightly", f.nightly, core.PausedByUnbind)
	f.expectStopped("ping", f.ping, core.PausedByUnbind)

	expectStatus(t, "rebind", f.bind("instance-2", "binding-2", "app-a"), nethttp.StatusCreated)
	f.expectRunning("nightly", f.nightly)
	f.expectRunning("ping", f.ping)
}

func TestProvisionFailsClosedWhenCloudControllerCannotAnswer(t *testing.T) {
	f := newFixture(t)

	expectStatus(t, "first", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	f.cf.FailServiceInstanceLookups(true)

	rec := f.provision("instance-2", "space-a")
	expectStatus(t, "second", rec, nethttp.StatusInternalServerError)
	if strings.Contains(rec.Body.String(), "lookup failed") {
		t.Errorf("the description should not carry the lookup error: %s", rec.Body.String())
	}

	if _, err := f.instances.Get("instance-1"); err != nil {
		t.Errorf("the first instance must stay, got %v", err)
	}
}

func TestProvisionHidesStoreErrors(t *testing.T) {
	f := newFixture(t)
	f.instances.Fail(fmt.Errorf("pq: password authentication failed for user scheduler"))

	rec := f.provision("instance-1", "space-a")
	expectStatus(t, "provision", rec, nethttp.StatusInternalServerError)
	if strings.Contains(rec.Body.String(), "pq:") {
		t.Errorf("the description should not carry the database error: %s", rec.Body.String())
	}
}

func TestBind(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)

	rec := f.bind("instance-1", "binding-1", "app-a")
	expectStatus(t, "bind", rec, nethttp.StatusCreated)

	want := map[string]any{"api_url": apiURL, "instance_id": "instance-1", "app_guid": "app-a"}
	credentials, _ := decode(t, rec)["credentials"].(map[string]any)
	if len(credentials) != len(want) {
		t.Fatalf("want credentials %v, got %v", want, credentials)
	}
	for key, value := range want {
		if credentials[key] != value {
			t.Errorf("credentials[%s] = %v, want %v", key, credentials[key], value)
		}
	}

	t.Run("repeating it identically answers 200 with the same credentials", func(t *testing.T) {
		rec := f.bind("instance-1", "binding-1", "app-a")
		expectStatus(t, "repeat", rec, nethttp.StatusOK)

		again, _ := decode(t, rec)["credentials"].(map[string]any)
		if again["app_guid"] != "app-a" || again["instance_id"] != "instance-1" {
			t.Errorf("unexpected credentials %v", again)
		}
	})

	t.Run("the same ID for another app answers 409", func(t *testing.T) {
		f.cf.PrepareApp("app-c", "space-a")
		expectStatus(t, "conflict", f.bind("instance-1", "binding-1", "app-c"), nethttp.StatusConflict)
	})

	t.Run("fetching it returns the credentials", func(t *testing.T) {
		rec := f.do(nethttp.MethodGet, "/v2/service_instances/instance-1/service_bindings/binding-1", "")
		expectStatus(t, "fetch", rec, nethttp.StatusOK)

		fetched, _ := decode(t, rec)["credentials"].(map[string]any)
		if fetched["app_guid"] != "app-a" {
			t.Errorf("unexpected credentials %v", fetched)
		}
	})

	t.Run("fetching it under another instance answers 404", func(t *testing.T) {
		expectStatus(t, "provision", f.provision("instance-2", "space-b"), nethttp.StatusCreated)
		rec := f.do(nethttp.MethodGet, "/v2/service_instances/instance-2/service_bindings/binding-1", "")
		expectStatus(t, "fetch", rec, nethttp.StatusNotFound)
	})

	t.Run("an app in another space answers 400", func(t *testing.T) {
		expectStatus(t, "cross-space", f.bind("instance-1", "binding-2", "app-b"), nethttp.StatusBadRequest)
		if _, err := f.bindings.Get("binding-2"); !errors.Is(err, core.ErrNotFound) {
			t.Errorf("binding-2 should not exist, got %v", err)
		}
	})

	t.Run("an unknown instance answers 404", func(t *testing.T) {
		expectStatus(t, "unknown instance", f.bind("nope", "binding-3", "app-a"), nethttp.StatusNotFound)
	})
}

func TestServiceKeysEnableNoApp(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)

	rec := f.bind("instance-1", "key-1", "")
	expectStatus(t, "service key", rec, nethttp.StatusCreated)

	credentials, _ := decode(t, rec)["credentials"].(map[string]any)
	if _, found := credentials["app_guid"]; found || credentials["instance_id"] != "instance-1" {
		t.Errorf("unexpected service key credentials %v", credentials)
	}

	bound, err := f.bindings.Bound("")
	if err != nil || bound {
		t.Errorf("a service key must not bind an app, got bound=%t err=%v", bound, err)
	}

	expectStatus(t, "delete key", f.unbind("instance-1", "key-1"), nethttp.StatusOK)
	f.expectRunning("nightly", f.nightly)
}

func TestUnbindPausesAndRebindResumesOnlyWhatUnbindPaused(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)

	expectStatus(t, "unbind", f.unbind("instance-1", "binding-1"), nethttp.StatusOK)

	f.expectStopped("nightly", f.nightly, core.PausedByUnbind)
	f.expectStopped("hourly", f.hourly, core.PausedByUnbind)
	f.expectStopped("ping", f.ping, core.PausedByUnbind)
	// The owner's disabled schedule keeps no mark, and it leaves cron too.
	f.expectStopped("owned", f.owned, "")

	if _, err := f.services.Jobs.Get(f.job.GUID); err != nil {
		t.Errorf("unbind must keep the job, got %v", err)
	}
	if _, err := f.services.Calls.Get(f.call.GUID); err != nil {
		t.Errorf("unbind must keep the call, got %v", err)
	}

	t.Run("unbinding again answers 410", func(t *testing.T) {
		expectStatus(t, "again", f.unbind("instance-1", "binding-1"), nethttp.StatusGone)
	})

	expectStatus(t, "rebind", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusCreated)

	f.expectRunning("nightly", f.nightly)
	f.expectRunning("hourly", f.hourly)
	f.expectRunning("ping", f.ping)
	f.expectStopped("owned", f.owned, "")
}

// A schedule the owner disables while the app is unbound is not touched by
// the rebind, because its mark was the unbind's.
func TestRebindLeavesAScheduleTheOwnerChangedAlone(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)
	expectStatus(t, "unbind", f.unbind("instance-1", "binding-1"), nethttp.StatusOK)

	added := f.schedule(f.job, false)

	expectStatus(t, "rebind", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusCreated)

	if stored := f.state(added); stored.Enabled {
		t.Error("a schedule created disabled after the unbind must stay disabled")
	}
	f.expectRunning("nightly", f.nightly)
}

func TestUnbindKeepsSchedulesWhileAnotherBindingRemains(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)
	expectStatus(t, "bind again", f.bind("instance-1", "binding-2", "app-a"), nethttp.StatusCreated)

	expectStatus(t, "unbind one", f.unbind("instance-1", "binding-1"), nethttp.StatusOK)

	f.expectRunning("nightly", f.nightly)
	f.expectRunning("ping", f.ping)

	expectStatus(t, "unbind the other", f.unbind("instance-1", "binding-2"), nethttp.StatusOK)

	f.expectStopped("nightly", f.nightly, core.PausedByUnbind)
	f.expectStopped("ping", f.ping, core.PausedByUnbind)
}

func TestUnbindLeavesTheBindingWhenTheStoreFails(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)

	f.bindings.Fail(fmt.Errorf("pq: connection refused"))
	rec := f.unbind("instance-1", "binding-1")
	expectStatus(t, "unbind", rec, nethttp.StatusInternalServerError)
	if strings.Contains(rec.Body.String(), "pq:") {
		t.Errorf("the description should not carry the database error: %s", rec.Body.String())
	}
	f.bindings.Fail(nil)

	if _, err := f.bindings.Get("binding-1"); err != nil {
		t.Errorf("the binding must stay for Cloud Controller to retry, got %v", err)
	}
}

func TestDeprovision(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)
	expectStatus(t, "key", f.bind("instance-1", "key-1", ""), nethttp.StatusCreated)

	expectStatus(t, "deprovision", f.deprovision("instance-1"), nethttp.StatusOK)

	if f.instances.Count() != 0 || f.bindings.Count() != 0 {
		t.Errorf("want no instances or bindings, got %d and %d", f.instances.Count(), f.bindings.Count())
	}
	f.expectStopped("nightly", f.nightly, core.PausedByUnbind)
	f.expectStopped("ping", f.ping, core.PausedByUnbind)

	t.Run("deprovisioning again answers 410", func(t *testing.T) {
		expectStatus(t, "again", f.deprovision("instance-1"), nethttp.StatusGone)
	})

	t.Run("a new instance and binding resume the schedules", func(t *testing.T) {
		expectStatus(t, "provision", f.provision("instance-2", "space-a"), nethttp.StatusCreated)
		expectStatus(t, "bind", f.bind("instance-2", "binding-2", "app-a"), nethttp.StatusCreated)
		f.expectRunning("nightly", f.nightly)
		f.expectRunning("ping", f.ping)
	})
}

func TestUpdate(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)

	path := "/v2/service_instances/instance-1"
	same := fmt.Sprintf(`{"service_id":%q,"plan_id":%q}`, broker.ServiceID, broker.PlanID)
	other := fmt.Sprintf(`{"service_id":%q,"plan_id":"other"}`, broker.ServiceID)

	expectStatus(t, "same plan", f.do(nethttp.MethodPatch, path, same), nethttp.StatusOK)
	expectStatus(t, "other plan", f.do(nethttp.MethodPatch, path, other), nethttp.StatusBadRequest)
	expectStatus(t, "unknown instance", f.do(nethttp.MethodPatch, "/v2/service_instances/nope", same), nethttp.StatusNotFound)
}

func TestNoOperationsToPoll(t *testing.T) {
	f := newFixture(t)
	expectStatus(t, "provision", f.provision("instance-1", "space-a"), nethttp.StatusCreated)
	expectStatus(t, "bind", f.bind("instance-1", "binding-1", "app-a"), nethttp.StatusCreated)

	expectStatus(t, "instance", f.do(nethttp.MethodGet, "/v2/service_instances/instance-1/last_operation", ""), nethttp.StatusBadRequest)
	expectStatus(t, "binding", f.do(nethttp.MethodGet, "/v2/service_instances/instance-1/service_bindings/binding-1/last_operation", ""), nethttp.StatusBadRequest)
}

func TestNewHandlerNeedsCredentials(t *testing.T) {
	services := &core.Services{
		Logger:    logger.New(),
		Instances: mock.NewInstanceService(),
		Bindings:  mock.NewBindingService(),
	}

	for name, config := range map[string]broker.Config{
		"no username": {Enabled: true, Password: brokerPassword, APIURL: apiURL},
		"no password": {Enabled: true, Username: brokerUser, APIURL: apiURL},
	} {
		t.Run(name, func(t *testing.T) {
			handler, err := broker.NewHandler(services, config)
			if err == nil || handler != nil {
				t.Fatal("want an error and no handler")
			}
		})
	}

	t.Run("no stores", func(t *testing.T) {
		_, err := broker.NewHandler(&core.Services{Logger: logger.New()}, broker.Config{Enabled: true, Username: brokerUser, Password: brokerPassword, APIURL: apiURL})
		if err == nil {
			t.Fatal("want an error")
		}
	})
}

// Without a broker handler, /v2 is not served at all.
func TestServerWithoutTheBroker(t *testing.T) {
	services := &core.Services{Logger: logger.New(), Auth: mock.NewAuthService()}
	server := schedhttp.Server("127.0.0.1:0", services, nil).Handler

	req := httptest.NewRequest(nethttp.MethodGet, "/v2/catalog", nil)
	req.Header.Set("X-Broker-API-Version", apiVersion)
	req.SetBasicAuth(brokerUser, brokerPassword)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusNotFound {
		t.Fatalf("got %d, want 404", rec.Code)
	}
}
