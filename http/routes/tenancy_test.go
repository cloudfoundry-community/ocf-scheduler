package routes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/cf"
	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/cron"
	"github.com/cloudfoundry-community/ocf-scheduler/logger"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

const tenantBSecret = "Basic dGVuYW50LWItc2VjcmV0"

// recordingRunner stands in for the CF task runner and keeps what it was
// asked to run.
type recordingRunner struct {
	locker sync.Mutex
	ran    []core.Executable
}

func (runner *recordingRunner) Execute(_ *core.Services, _ *core.Execution, executable core.Executable) {
	runner.locker.Lock()
	defer runner.locker.Unlock()

	runner.ran = append(runner.ran, executable)
}

func (runner *recordingRunner) count() int {
	runner.locker.Lock()
	defer runner.locker.Unlock()

	return len(runner.ran)
}

// tenancy is two tenants, A and B, each with an app, a job, a call, and a
// schedule on each, served through the real cf.AuthService.
type tenancy struct {
	e        *echo.Echo
	services *core.Services
	cf       *mock.CFClient
	runner   *recordingRunner

	jobA, jobB            *core.Job
	callA, callB          *core.Call
	jobSchedB, callSchedB *core.Schedule

	userA, userB, admin, manager, auditor string
}

// token builds an Authorization value whose JWT carries the given scopes.
func token(name string, scopes ...string) string {
	payload, _ := json.Marshal(map[string]any{"scope": scopes, "user_name": name})
	return "Bearer " +
		base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".sig-" + name
}

func newTenancy(t *testing.T) *tenancy {
	t.Helper()

	client, err := mock.NewCFClient()
	if err != nil {
		t.Fatal(err)
	}
	client.PrepareApp("app-a", "space-a")
	client.PrepareApp("app-b", "space-b")
	client.GrantRole("user-a", "space-a", "space_developer")
	client.GrantRole("user-b", "space-b", "space_developer")
	client.GrantRole("user-m", "space-b", "space_manager")
	client.GrantRole("user-r", "space-b", "space_auditor")

	tn := &tenancy{
		cf:      client,
		runner:  &recordingRunner{},
		userA:   token("user-a", "openid", "cloud_controller.read", "cloud_controller.write"),
		userB:   token("user-b", "openid", "cloud_controller.read", "cloud_controller.write"),
		manager: token("user-m", "openid", "cloud_controller.read"),
		auditor: token("user-r", "openid", "cloud_controller.read"),
		admin:   token("admin", "openid", "cloud_controller.admin"),
	}

	// Stand in for UAA's userinfo, which knows each token's user.
	users := map[string]string{}
	for user, auth := range map[string]string{
		"user-a": tn.userA, "user-b": tn.userB, "user-m": tn.manager, "user-r": tn.auditor, "admin": tn.admin,
	} {
		users[strings.TrimPrefix(auth, "Bearer ")] = user
	}
	identify := func(bearer string) (*cf.Identity, error) {
		user, found := users[bearer]
		if !found {
			return nil, fmt.Errorf("invalid token")
		}
		return &cf.Identity{UserGUID: user, Username: user}, nil
	}

	log := logger.New()
	jobs := mock.NewJobService()
	calls := mock.NewCallService()
	schedules := mock.NewScheduleService()
	schedules.Owners(jobs, calls)
	tn.services = &core.Services{
		Info:       cf.NewInfoService(client),
		Jobs:       jobs,
		Calls:      calls,
		Schedules:  schedules,
		Executions: mock.NewExecutionService(),
		Runner:     tn.runner,
		Cron:       cron.NewCronService(log, cron.Rules{}),
		Logger:     log,
		Auth:       cf.NewAuthServiceWithIdentity(client, log, identify),
	}

	tn.e = echo.New()
	Apply(tn.e, tn.services)

	tn.jobA = tn.persistJob(t, "a")
	tn.jobB = tn.persistJob(t, "b")
	tn.callA = tn.persistCall(t, "a")
	tn.callB = tn.persistCall(t, "b")
	tn.jobSchedB = tn.persistSchedule(t, tn.jobB)
	tn.callSchedB = tn.persistSchedule(t, tn.callB)

	return tn
}

func (tn *tenancy) persistJob(t *testing.T, tenant string) *core.Job {
	t.Helper()
	job, err := tn.services.Jobs.Persist(&core.Job{Name: "job-" + tenant, Command: "true", AppGUID: "app-" + tenant, SpaceGUID: "space-" + tenant})
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func (tn *tenancy) persistCall(t *testing.T, tenant string) *core.Call {
	t.Helper()
	call, err := tn.services.Calls.Persist(&core.Call{Name: "call-" + tenant, URL: "https://example.com/" + tenant, AuthHeader: tenantBSecret, AppGUID: "app-" + tenant, SpaceGUID: "space-" + tenant})
	if err != nil {
		t.Fatal(err)
	}
	return call
}

// persistSchedule stores a schedule and registers it with cron, as the
// create route does, so the delete routes can unschedule it.
func (tn *tenancy) persistSchedule(t *testing.T, executable core.Executable) *core.Schedule {
	t.Helper()
	schedule := &core.Schedule{Enabled: true, Expression: "0 2 * * *", ExpressionType: "cron_expression", RefType: executable.Type()}
	var run core.Runnable
	if job, err := executable.ToJob(); err == nil {
		schedule.RefGUID = job.GUID
		run = core.NewJobRun(job, schedule, tn.services)
	} else {
		call, _ := executable.ToCall()
		schedule.RefGUID = call.GUID
		run = core.NewCallRun(call, schedule, tn.services)
	}
	if _, err := tn.services.Schedules.Persist(schedule); err != nil {
		t.Fatal(err)
	}
	if err := tn.services.Cron.Add(run); err != nil {
		t.Fatal(err)
	}
	return schedule
}

func (tn *tenancy) do(method, path, body, auth string) *httptest.ResponseRecorder {
	return tn.serve(method, path, strings.NewReader(body), auth)
}

// serve sends a request whose body comes from body as the route reads it.
func (tn *tenancy) serve(method, path string, body io.Reader, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if auth != "" {
		req.Header.Set(echo.HeaderAuthorization, auth)
	}
	rec := httptest.NewRecorder()
	tn.e.ServeHTTP(rec, req)
	return rec
}

// Route kinds decide the denied status. Routes that name an app or a space
// answer 403, and routes that name a job or call answer 404. The validate
// route first checks that the caller holds a role anywhere, as it did before.
const (
	scoped = iota
	object
	validate
)

type request struct {
	method, path, body string
	ok                 int
	kind               int
}

type expectation func(request) int

// denied answers scopedStatus, objectStatus, or validateStatus by route kind.
func denied(scopedStatus, objectStatus, validateStatus int) expectation {
	return func(r request) int {
		switch r.kind {
		case scoped:
			return scopedStatus
		case validate:
			return validateStatus
		default:
			return objectStatus
		}
	}
}

// allowed answers each route's own success status.
func allowed(r request) int {
	return r.ok
}

func (tn *tenancy) check(t *testing.T, auth string, requests []request, want expectation) {
	t.Helper()
	for _, r := range requests {
		rec := tn.do(r.method, r.path, r.body, auth)
		if code := want(r); rec.Code != code {
			t.Errorf("%s %s: status %d, want %d, body %s", r.method, r.path, rec.Code, code, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "auth_header") || strings.Contains(rec.Body.String(), tenantBSecret) {
			t.Errorf("%s %s: response carries an auth header: %s", r.method, r.path, rec.Body)
		}
	}
}

// requestsOnB covers every route that reaches tenant B's app, space, jobs,
// calls, or schedules, ending with the deletes.
func (tn *tenancy) requestsOnB() []request {
	job, call := "/jobs/"+tn.jobB.GUID, "/calls/"+tn.callB.GUID
	schedule := `{"enabled": true, "expression": "0 3 * * *", "expression_type": "cron_expression"}`
	return []request{
		{http.MethodPost, "/jobs?app_guid=app-b", `{"name": "intruder", "command": "env"}`, http.StatusCreated, scoped},
		{http.MethodPost, "/calls?app_guid=app-b", `{"name": "intruder", "url": "https://example.com", "auth_header": "x"}`, http.StatusCreated, scoped},
		{http.MethodGet, "/jobs?space_guid=space-b", "", http.StatusOK, scoped},
		{http.MethodGet, "/calls?space_guid=space-b", "", http.StatusOK, scoped},
		{http.MethodGet, job, "", http.StatusOK, object},
		{http.MethodGet, call, "", http.StatusOK, object},
		{http.MethodPost, job + "/execute", `{}`, http.StatusCreated, object},
		{http.MethodPost, call + "/execute", `{}`, http.StatusCreated, object},
		{http.MethodGet, job + "/history", "", http.StatusOK, object},
		{http.MethodGet, call + "/history", "", http.StatusOK, object},
		{http.MethodGet, job + "/schedules", "", http.StatusOK, object},
		{http.MethodGet, call + "/schedules", "", http.StatusOK, object},
		{http.MethodPost, job + "/schedules", schedule, http.StatusCreated, object},
		{http.MethodPost, call + "/schedules", schedule, http.StatusCreated, object},
		{http.MethodGet, job + "/schedules/" + tn.jobSchedB.GUID + "/history", "", http.StatusOK, object},
		{http.MethodGet, call + "/schedules/" + tn.callSchedB.GUID + "/history", "", http.StatusOK, object},
		{http.MethodPost, "/schedules/validate", `{"ref_type": "job", "ref_guid": "` + tn.jobB.GUID + `"}`, http.StatusOK, validate},
		{http.MethodPost, "/schedules/validate", `{"ref_type": "call", "ref_guid": "` + tn.callB.GUID + `"}`, http.StatusOK, validate},
		{http.MethodDelete, job + "/schedules/" + tn.jobSchedB.GUID, "", http.StatusNoContent, object},
		{http.MethodDelete, call + "/schedules/" + tn.callSchedB.GUID, "", http.StatusNoContent, object},
		{http.MethodDelete, job, "", http.StatusNoContent, object},
		{http.MethodDelete, call, "", http.StatusNoContent, object},
	}
}

func (tn *tenancy) assertBUntouched(t *testing.T) {
	t.Helper()
	if _, err := tn.services.Jobs.Get(tn.jobB.GUID); err != nil {
		t.Errorf("job B was deleted")
	}
	if _, err := tn.services.Calls.Get(tn.callB.GUID); err != nil {
		t.Errorf("call B was deleted")
	}
	if _, err := tn.services.Schedules.Get(tn.jobSchedB.GUID); err != nil {
		t.Errorf("job B's schedule was deleted")
	}
	if _, err := tn.services.Schedules.Get(tn.callSchedB.GUID); err != nil {
		t.Errorf("call B's schedule was deleted")
	}
	if jobs, _ := tn.services.Jobs.InSpace("space-b"); len(jobs) != 1 {
		t.Errorf("space B has %d jobs, want 1", len(jobs))
	}
	if calls := tn.services.Calls.InSpace("space-b"); len(calls) != 1 {
		t.Errorf("space B has %d calls, want 1", len(calls))
	}
	jobScheds, _ := tn.services.Schedules.ByJob(tn.jobB)
	callScheds, _ := tn.services.Schedules.ByCall(tn.callB)
	if n := len(jobScheds) + len(callScheds); n != 2 {
		t.Errorf("B has %d schedules, want 2", n)
	}
	if n := tn.runner.count(); n != 0 {
		t.Errorf("%d runs were dispatched, want 0", n)
	}
}

func TestDeveloperInAnotherSpaceIsDenied(t *testing.T) {
	tn := newTenancy(t)
	tn.check(t, tn.userA, tn.requestsOnB(), denied(http.StatusForbidden, http.StatusNotFound, http.StatusNotFound))
	tn.assertBUntouched(t)
}

func TestListingWithoutASpaceIsDenied(t *testing.T) {
	tn := newTenancy(t)
	tn.check(t, tn.userA, []request{
		{http.MethodGet, "/jobs?space_guid=", "", http.StatusOK, scoped},
		{http.MethodGet, "/calls", "", http.StatusOK, scoped},
	}, denied(http.StatusForbidden, http.StatusNotFound, http.StatusNotFound))
}

// A schedule GUID from tenant B, under a job or call tenant A owns, is not
// found, so A cannot reach B's schedules through A's own job or call.
func TestScheduleMustBelongToTheNamedJobOrCall(t *testing.T) {
	tn := newTenancy(t)
	tn.check(t, tn.userA, []request{
		{http.MethodGet, "/jobs/" + tn.jobA.GUID + "/schedules/" + tn.jobSchedB.GUID + "/history", "", http.StatusOK, object},
		{http.MethodGet, "/calls/" + tn.callA.GUID + "/schedules/" + tn.callSchedB.GUID + "/history", "", http.StatusOK, object},
		{http.MethodDelete, "/jobs/" + tn.jobA.GUID + "/schedules/" + tn.jobSchedB.GUID, "", http.StatusNoContent, object},
		{http.MethodDelete, "/calls/" + tn.callA.GUID + "/schedules/" + tn.callSchedB.GUID, "", http.StatusNoContent, object},
		{http.MethodDelete, "/jobs/" + tn.jobA.GUID + "/schedules/" + tn.callSchedB.GUID, "", http.StatusNoContent, object},
	}, denied(http.StatusForbidden, http.StatusNotFound, http.StatusNotFound))
	tn.assertBUntouched(t)
}

func TestSpaceManagerAndAuditorAreDenied(t *testing.T) {
	tn := newTenancy(t)
	tn.check(t, tn.manager, tn.requestsOnB(), denied(http.StatusForbidden, http.StatusNotFound, http.StatusNotFound))
	// An auditor holds no role that the old any-space check accepted either,
	// so the validate route stops it at that first check.
	tn.check(t, tn.auditor, tn.requestsOnB(), denied(http.StatusForbidden, http.StatusNotFound, http.StatusUnauthorized))
	tn.assertBUntouched(t)
}

func TestFailedRoleLookupIsDenied(t *testing.T) {
	tn := newTenancy(t)
	tn.cf.FailRoleLookups(true)
	tn.check(t, tn.userB, tn.requestsOnB(), denied(http.StatusForbidden, http.StatusNotFound, http.StatusUnauthorized))
	tn.assertBUntouched(t)
}

func TestUnauthenticatedCallerGets401(t *testing.T) {
	tn := newTenancy(t)
	for _, auth := range []string{"", "Bearer not-a-token", "Basic dXNlcjpwYXNz"} {
		tn.check(t, auth, tn.requestsOnB(), denied(http.StatusUnauthorized, http.StatusUnauthorized, http.StatusUnauthorized))
		tn.check(t, auth, []request{
			{http.MethodGet, "/jobs/no-such-job", "", http.StatusOK, object},
			{http.MethodGet, "/calls/no-such-call", "", http.StatusOK, object},
			{http.MethodPost, "/jobs?app_guid=", `{}`, http.StatusCreated, scoped},
		}, denied(http.StatusUnauthorized, http.StatusUnauthorized, http.StatusUnauthorized))
	}
	tn.assertBUntouched(t)
}

func TestSpaceDeveloperIsAllowedInOwnSpace(t *testing.T) {
	tn := newTenancy(t)
	tn.check(t, tn.userB, tn.requestsOnB(), allowed)
	if n := tn.runner.count(); n != 2 {
		t.Errorf("%d runs were dispatched, want 2", n)
	}
	if jobs, _ := tn.services.Jobs.InSpace("space-b"); len(jobs) != 1 || jobs[0].Name != "intruder" {
		t.Errorf("space B has %d jobs after the run, want only the new one", len(jobs))
	}
}

// Admins skip the role lookup, so they pass even while it fails.
func TestAdminIsAllowedInEverySpace(t *testing.T) {
	tn := newTenancy(t)
	tn.cf.FailRoleLookups(true)
	tn.check(t, tn.admin, tn.requestsOnB(), allowed)
	if n := tn.runner.count(); n != 2 {
		t.Errorf("%d runs were dispatched, want 2", n)
	}
}

func TestMissingThingsAre404ForAuthenticatedCallers(t *testing.T) {
	tn := newTenancy(t)
	tn.check(t, tn.userA, []request{
		{http.MethodGet, "/jobs/no-such-job", "", http.StatusNotFound, object},
		{http.MethodDelete, "/calls/no-such-call", "", http.StatusNotFound, object},
		{http.MethodGet, "/jobs/" + tn.jobA.GUID + "/schedules/no-such-schedule/history", "", http.StatusNotFound, object},
		{http.MethodPost, "/jobs?app_guid=", `{}`, http.StatusUnprocessableEntity, scoped},
	}, allowed)
}

// Every route that returns a call leaves out its auth header, for the owner
// and for an admin, while the scheduler still stores it and runs with it.
func TestAuthHeaderIsNeverReturned(t *testing.T) {
	tn := newTenancy(t)
	for i, auth := range []string{tn.userB, tn.admin} {
		name := fmt.Sprintf("hook-%d", i)
		rec := tn.do(http.MethodPost, "/calls?app_guid=app-b",
			`{"name": "`+name+`", "url": "https://example.com/hook", "auth_header": "`+tenantBSecret+`"}`, auth)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create: status %d, body %s", rec.Code, rec.Body)
		}
		var created struct {
			GUID string `json:"guid"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil || created.GUID == "" {
			t.Fatalf("create: body %s", rec.Body)
		}

		stored, err := tn.services.Calls.Get(created.GUID)
		if err != nil || stored.AuthHeader != tenantBSecret {
			t.Errorf("the call's auth header was not stored")
		}

		call := "/calls/" + created.GUID
		for _, r := range []request{
			{http.MethodPost, "/calls?app_guid=app-b", `{"name": "` + name + `-again", "url": "https://example.com", "auth_header": "` + tenantBSecret + `"}`, http.StatusCreated, scoped},
			{http.MethodGet, call, "", http.StatusOK, object},
			{http.MethodGet, "/calls?space_guid=space-b", "", http.StatusOK, object},
			{http.MethodPost, call + "/schedules", `{"enabled": true, "expression": "0 3 * * *", "expression_type": "cron_expression"}`, http.StatusCreated, object},
			{http.MethodGet, call + "/schedules", "", http.StatusOK, object},
			{http.MethodPost, call + "/execute", `{}`, http.StatusCreated, object},
			{http.MethodGet, call + "/history", "", http.StatusOK, object},
			{http.MethodPost, "/schedules/validate", `{"ref_type": "call", "ref_guid": "` + created.GUID + `"}`, http.StatusOK, object},
		} {
			tn.check(t, auth, []request{r}, allowed)
		}
	}

	tn.runner.locker.Lock()
	defer tn.runner.locker.Unlock()
	if len(tn.runner.ran) != 2 {
		t.Fatalf("%d runs were dispatched, want 2", len(tn.runner.ran))
	}
	for _, executable := range tn.runner.ran {
		call, err := executable.ToCall()
		if err != nil || call.AuthHeader != tenantBSecret {
			t.Errorf("the run did not get the stored auth header")
		}
	}
}
