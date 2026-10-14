package routes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudfoundry/go-cfclient/v3/resource"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// countingInfo counts the app lookups that reach Cloud Controller.
type countingInfo struct {
	core.InfoService
	locker sync.Mutex
	asked  map[string]int
}

func (info *countingInfo) AppExists(ctx context.Context, guid string) (bool, error) {
	info.locker.Lock()
	info.asked[guid]++
	info.locker.Unlock()

	return info.InfoService.AppExists(ctx, guid)
}

func (tn *tenancy) listOrphans(t *testing.T) []*orphan {
	t.Helper()
	rec := tn.do(http.MethodGet, "/orphans", "", tn.admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: status %d, body %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "auth_header") || strings.Contains(rec.Body.String(), tenantBSecret) {
		t.Errorf("list: the response carries an auth header: %s", rec.Body)
	}
	var output orphanCollection
	if err := json.Unmarshal(rec.Body.Bytes(), &output); err != nil {
		t.Fatalf("list: %v, body %s", err, rec.Body)
	}
	return output.Resources
}

func (tn *tenancy) deleteOrphans(t *testing.T, guids ...string) map[string]*orphanResult {
	t.Helper()
	body, _ := json.Marshal(orphanDeletion{GUIDs: guids})
	rec := tn.do(http.MethodPost, "/orphans/delete", string(body), tn.admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: status %d, body %s", rec.Code, rec.Body)
	}
	var output orphanResults
	if err := json.Unmarshal(rec.Body.Bytes(), &output); err != nil {
		t.Fatalf("delete: %v, body %s", err, rec.Body)
	}
	results := map[string]*orphanResult{}
	for _, result := range output.Results {
		results[result.GUID] = result
	}
	if len(results) != len(output.Results) {
		t.Errorf("delete: a GUID was answered more than once: %s", rec.Body)
	}
	return results
}

func expectResult(t *testing.T, results map[string]*orphanResult, guid, typ, want string) {
	t.Helper()
	result, found := results[guid]
	if !found {
		t.Errorf("%s: no result", guid)
		return
	}
	if result.Result != want || result.Type != typ {
		t.Errorf("%s: result %q for type %q, want %q for %q (%s)", guid, result.Result, result.Type, want, typ, result.Message)
	}
}

func TestOrphanRoutesRefuseNonAdmins(t *testing.T) {
	tn := newTenancy(t)
	tn.cf.RemoveApp("app-b")
	requests := []request{
		{http.MethodGet, "/orphans", "", http.StatusOK, scoped},
		{http.MethodPost, "/orphans/delete", `{"guids": ["` + tn.jobB.GUID + `", "` + tn.callB.GUID + `"]}`, http.StatusOK, scoped},
	}

	for _, auth := range []string{tn.userA, tn.userB, tn.manager, tn.auditor} {
		tn.check(t, auth, requests, denied(http.StatusForbidden, http.StatusForbidden, http.StatusForbidden))
	}
	for _, auth := range []string{"", "Bearer not-a-token", "Basic dXNlcjpwYXNz"} {
		tn.check(t, auth, requests, denied(http.StatusUnauthorized, http.StatusUnauthorized, http.StatusUnauthorized))
	}
	tn.cf.FailRoleLookups(true)
	tn.check(t, tn.userB, requests, denied(http.StatusForbidden, http.StatusForbidden, http.StatusForbidden))

	tn.assertBUntouched(t)
}

func TestListOrphansShowsOnlyJobsAndCallsWhoseAppIsGone(t *testing.T) {
	tn := newTenancy(t)
	info := &countingInfo{InfoService: tn.services.Info, asked: map[string]int{}}
	tn.services.Info = info

	// A second job on app B, so app B has three things to check.
	extra, err := tn.services.Jobs.Persist(&core.Job{Name: "job-b-2", Command: "true", AppGUID: "app-b", SpaceGUID: "space-b"})
	if err != nil {
		t.Fatal(err)
	}
	disabled := &core.Schedule{Enabled: false, Expression: "0 4 * * *", ExpressionType: "cron_expression", RefType: "job", RefGUID: extra.GUID}
	if _, err := tn.services.Schedules.Persist(disabled); err != nil {
		t.Fatal(err)
	}

	if orphans := tn.listOrphans(t); len(orphans) != 0 {
		t.Errorf("listed %d orphans while every app exists, want none", len(orphans))
	}

	tn.cf.RemoveApp("app-b")
	info.asked = map[string]int{}

	orphans := tn.listOrphans(t)
	listed := map[string]*orphan{}
	for _, o := range orphans {
		listed[o.GUID] = o
	}
	if len(orphans) != 3 || len(listed) != 3 {
		t.Fatalf("listed %d orphans, want job B, the second job on app B, and call B", len(orphans))
	}

	for _, want := range []struct {
		guid, typ, name string
		enabled         bool
		schedule        string
	}{
		{tn.jobB.GUID, "job", "job-b", true, tn.jobSchedB.GUID},
		{extra.GUID, "job", "job-b-2", false, disabled.GUID},
		{tn.callB.GUID, "call", "call-b", true, tn.callSchedB.GUID},
	} {
		o := listed[want.guid]
		if o == nil {
			t.Errorf("%s %s is not listed", want.typ, want.name)
			continue
		}
		if o.Type != want.typ || o.Name != want.name || o.AppGUID != "app-b" || o.SpaceGUID != "space-b" {
			t.Errorf("%s is listed as %+v", want.name, o)
		}
		if len(o.Schedules) != 1 || o.Schedules[0].GUID != want.schedule || o.Schedules[0].Enabled != want.enabled {
			t.Errorf("%s is listed with schedules %+v, want %s enabled %t", want.name, o.Schedules, want.schedule, want.enabled)
		}
	}

	for _, guid := range []string{tn.jobA.GUID, tn.callA.GUID} {
		if listed[guid] != nil {
			t.Errorf("%s is listed, but its app exists", guid)
		}
	}

	if info.asked["app-a"] != 1 || info.asked["app-b"] != 1 || len(info.asked) != 2 {
		t.Errorf("app lookups %v, want one for each app", info.asked)
	}

	// Listing deletes nothing.
	if jobs, _ := tn.services.Jobs.InSpace("space-b"); len(jobs) != 2 {
		t.Errorf("space B has %d jobs after the list, want 2", len(jobs))
	}
	if n := tn.services.Cron.Count(); n != 2 {
		t.Errorf("cron holds %d entries after the list, want 2", n)
	}
}

// A lookup that fails for any reason but not-found fails the whole list,
// even when other apps answered not-found.
func TestListOrphansFailsWhenAnAppLookupFails(t *testing.T) {
	for name, failure := range map[string]error{
		"403": resource.NewNotAuthorizedError(),
		"500": resource.CloudFoundryHTTPError{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error"},
		"503": resource.CloudFoundryHTTPError{StatusCode: http.StatusServiceUnavailable, Status: "503 Service Unavailable"},
	} {
		t.Run(name, func(t *testing.T) {
			tn := newTenancy(t)
			tn.cf.RemoveApp("app-b")
			tn.cf.FailAppLookup("app-a", failure)

			rec := tn.do(http.MethodGet, "/orphans", "", tn.admin)
			if rec.Code != http.StatusBadGateway {
				t.Errorf("status %d, want %d, body %s", rec.Code, http.StatusBadGateway, rec.Body)
			}
			for _, guid := range []string{tn.jobB.GUID, tn.callB.GUID, "resources"} {
				if strings.Contains(rec.Body.String(), guid) {
					t.Errorf("the failed list carries %s: %s", guid, rec.Body)
				}
			}
		})
	}
}

func TestDeleteOrphansRemovesSchedulesFromCronAndTheDatabase(t *testing.T) {
	tn := newTenancy(t)
	tn.cf.RemoveApp("app-b")

	if n := tn.services.Cron.Count(); n != 2 {
		t.Fatalf("cron holds %d entries before the delete, want 2", n)
	}

	results := tn.deleteOrphans(t, tn.jobB.GUID, tn.callB.GUID, tn.jobA.GUID, "no-such-guid", tn.jobB.GUID)
	if len(results) != 4 {
		t.Errorf("got %d results, want one for each distinct GUID", len(results))
	}
	expectResult(t, results, tn.jobB.GUID, "job", orphanDeleted)
	expectResult(t, results, tn.callB.GUID, "call", orphanDeleted)
	expectResult(t, results, tn.jobA.GUID, "job", orphanAppExists)
	expectResult(t, results, "no-such-guid", "", orphanNotFound)

	if _, err := tn.services.Jobs.Get(tn.jobB.GUID); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("job B is still stored")
	}
	if _, err := tn.services.Calls.Get(tn.callB.GUID); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("call B is still stored")
	}
	for _, schedule := range []*core.Schedule{tn.jobSchedB, tn.callSchedB} {
		if _, err := tn.services.Schedules.Get(schedule.GUID); err == nil {
			t.Errorf("schedule %s is still stored", schedule.GUID)
		}
	}

	if n := tn.services.Cron.Count(); n != 0 {
		t.Errorf("cron holds %d entries after the delete, want 0", n)
	}
	for _, run := range []core.Runnable{
		core.NewJobRun(tn.jobB, tn.jobSchedB, tn.services),
		core.NewCallRun(tn.callB, tn.callSchedB, tn.services),
	} {
		if err := tn.services.Cron.Delete(run); !errors.Is(err, core.ErrNotScheduled) {
			t.Errorf("schedule %s is still in cron", run.Schedule().GUID)
		}
	}

	if _, err := tn.services.Jobs.Get(tn.jobA.GUID); err != nil {
		t.Errorf("job A was deleted, but its app exists")
	}

	// Deleting them again finds nothing.
	results = tn.deleteOrphans(t, tn.jobB.GUID, tn.callB.GUID)
	expectResult(t, results, tn.jobB.GUID, "", orphanNotFound)
	expectResult(t, results, tn.callB.GUID, "", orphanNotFound)
}

// The delete asks Cloud Controller again, so an app that answers again
// since the list keeps its jobs and calls.
func TestDeleteOrphansRefusesAnAppThatCameBack(t *testing.T) {
	tn := newTenancy(t)
	tn.cf.RemoveApp("app-b")
	if orphans := tn.listOrphans(t); len(orphans) != 2 {
		t.Fatalf("listed %d orphans, want job B and call B", len(orphans))
	}

	tn.cf.RestoreApp("app-b")

	results := tn.deleteOrphans(t, tn.jobB.GUID, tn.callB.GUID)
	expectResult(t, results, tn.jobB.GUID, "job", orphanAppExists)
	expectResult(t, results, tn.callB.GUID, "call", orphanAppExists)
	tn.assertBUntouched(t)
	if n := tn.services.Cron.Count(); n != 2 {
		t.Errorf("cron holds %d entries, want 2", n)
	}
}

func TestDeleteOrphansRefusesWhenTheCheckFails(t *testing.T) {
	for name, failure := range map[string]error{
		"403":     resource.NewNotAuthorizedError(),
		"502":     resource.CloudFoundryHTTPError{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway"},
		"bare404": resource.CloudFoundryHTTPError{StatusCode: http.StatusNotFound, Status: "404 Not Found"},
		"timeout": context.DeadlineExceeded,
	} {
		t.Run(name, func(t *testing.T) {
			tn := newTenancy(t)
			tn.cf.FailAppLookup("app-b", failure)

			results := tn.deleteOrphans(t, tn.jobB.GUID, tn.callB.GUID)
			expectResult(t, results, tn.jobB.GUID, "job", orphanFailed)
			expectResult(t, results, tn.callB.GUID, "call", orphanFailed)
			tn.assertBUntouched(t)
			if n := tn.services.Cron.Count(); n != 2 {
				t.Errorf("cron holds %d entries, want 2", n)
			}
		})
	}
}

func TestDeleteOrphansNeedsAList(t *testing.T) {
	tn := newTenancy(t)
	tn.cf.RemoveApp("app-b")
	for _, body := range []string{`{"guids": []}`, `{}`, `{"guids": null}`, `{"guids": `, `[]`} {
		rec := tn.do(http.MethodPost, "/orphans/delete", body, tn.admin)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %s: status %d, want %d, body %s", body, rec.Code, http.StatusBadRequest, rec.Body)
		}
	}
	tn.assertBUntouched(t)
}

// expectNothingAsked checks that a refused request listed and deleted
// nothing, and never asked Cloud Controller about an app.
func expectNothingAsked(t *testing.T, tn *tenancy, info *countingInfo, rec *httptest.ResponseRecorder) {
	t.Helper()
	if len(info.asked) != 0 {
		t.Errorf("asked Cloud Controller about apps %v", info.asked)
	}
	for _, guid := range []string{tn.jobB.GUID, tn.callB.GUID, "resources", "results"} {
		if strings.Contains(rec.Body.String(), guid) {
			t.Errorf("the refusal carries %s: %s", guid, rec.Body)
		}
	}
	tn.assertBUntouched(t)
	if n := tn.services.Cron.Count(); n != 2 {
		t.Errorf("cron holds %d entries, want 2", n)
	}
}

// The scopes that let the scheduler's own client read every app, written
// out here so the test does not lean on the list it checks.
var wantAppReaderScopes = []string{"cloud_controller.admin", "cloud_controller.admin_read_only", "cloud_controller.global_auditor"}

func TestOrphanRoutesWorkWithEachAppReaderScope(t *testing.T) {
	for _, scope := range wantAppReaderScopes {
		t.Run(scope, func(t *testing.T) {
			tn := newTenancy(t)
			tn.cf.SetClientScopes("uaa.resource", scope)
			tn.cf.RemoveApp("app-b")

			if orphans := tn.listOrphans(t); len(orphans) != 2 {
				t.Errorf("listed %d orphans, want job B and call B", len(orphans))
			}

			results := tn.deleteOrphans(t, tn.jobB.GUID, tn.callB.GUID)
			expectResult(t, results, tn.jobB.GUID, "job", orphanDeleted)
			expectResult(t, results, tn.callB.GUID, "call", orphanDeleted)
		})
	}
}

func TestOrphanRoutesRefuseWithoutAnAppReaderScope(t *testing.T) {
	for name, scopes := range map[string][]string{
		"no scopes":         nil,
		"read and write":    {"cloud_controller.read", "cloud_controller.write", "uaa.resource"},
		"lookalike scopes":  {"cloud_controller.admin_read", "cloud_controller.global_auditors"},
		"another admin":     {"uaa.admin", "scim.read"},
		"an empty scope":    {""},
		"auditor substring": {"x.cloud_controller.global_auditor"},
	} {
		t.Run(name, func(t *testing.T) {
			for _, req := range []struct{ method, path, body string }{
				{http.MethodGet, "/orphans", ""},
				{http.MethodPost, "/orphans/delete", ""},
			} {
				tn := newTenancy(t)
				info := &countingInfo{InfoService: tn.services.Info, asked: map[string]int{}}
				tn.services.Info = info
				tn.cf.SetClientScopes(scopes...)
				tn.cf.RemoveApp("app-b")

				body := req.body
				if req.method == http.MethodPost {
					body = `{"guids": ["` + tn.jobB.GUID + `", "` + tn.callB.GUID + `"]}`
				}

				rec := tn.do(req.method, req.path, body, tn.admin)
				if rec.Code != http.StatusInternalServerError {
					t.Errorf("%s %s: status %d, want %d, body %s", req.method, req.path, rec.Code, http.StatusInternalServerError, rec.Body)
				}
				for _, scope := range wantAppReaderScopes {
					if !strings.Contains(rec.Body.String(), scope) {
						t.Errorf("%s %s: the refusal does not name %s: %s", req.method, req.path, scope, rec.Body)
					}
				}
				expectNothingAsked(t, tn, info, rec)
			}
		})
	}
}

func TestOrphanRoutesRefuseWhenTheClientScopesCannotBeRead(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	for name, spoil := range map[string]func(tn *tenancy){
		"UAA unreachable": func(tn *tenancy) { tn.cf.FailClientToken(errors.New("connection refused")) },
		"empty token":     func(tn *tenancy) { tn.cf.SetClientToken("") },
		"not a JWT":       func(tn *tenancy) { tn.cf.SetClientToken("opaque-token") },
		"payload not b64": func(tn *tenancy) { tn.cf.SetClientToken(header + ".!!!.sig") },
		"payload not JSON": func(tn *tenancy) {
			tn.cf.SetClientToken(header + "." + base64.RawURLEncoding.EncodeToString([]byte("nope")) + ".sig")
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, req := range []struct{ method, path string }{
				{http.MethodGet, "/orphans"},
				{http.MethodPost, "/orphans/delete"},
			} {
				tn := newTenancy(t)
				info := &countingInfo{InfoService: tn.services.Info, asked: map[string]int{}}
				tn.services.Info = info
				spoil(tn)
				tn.cf.RemoveApp("app-b")

				body := ""
				if req.method == http.MethodPost {
					body = `{"guids": ["` + tn.jobB.GUID + `", "` + tn.callB.GUID + `"]}`
				}

				rec := tn.do(req.method, req.path, body, tn.admin)
				if rec.Code != http.StatusBadGateway {
					t.Errorf("%s %s: status %d, want %d, body %s", req.method, req.path, rec.Code, http.StatusBadGateway, rec.Body)
				}
				if !strings.Contains(rec.Body.String(), "scopes") {
					t.Errorf("%s %s: the refusal does not say the scopes could not be read: %s", req.method, req.path, rec.Body)
				}
				expectNothingAsked(t, tn, info, rec)
			}
		})
	}
}

// vanishingJobs deletes the job and then reports it as not found, as the
// slower of two concurrent deletes would see it.
type vanishingJobs struct {
	core.JobService
}

func (jobs vanishingJobs) Delete(job *core.Job) error {
	if err := jobs.JobService.Delete(job); err != nil {
		return err
	}

	return fmt.Errorf("job with GUID %s not found: %w", job.GUID, core.ErrNotFound)
}

func TestDeleteOrphansAnswersNotFoundWhenTheJobIsAlreadyGone(t *testing.T) {
	tn := newTenancy(t)
	tn.cf.RemoveApp("app-b")
	tn.services.Jobs = vanishingJobs{tn.services.Jobs}

	results := tn.deleteOrphans(t, tn.jobB.GUID)
	expectResult(t, results, tn.jobB.GUID, "job", orphanNotFound)
}

// gatedInfo holds each app lookup until enough lookups are in flight at
// once, or until wait passes, so a list that asks one app at a time shows up
// as a slow list that never had two in flight. The lookup for fail waits the
// same way and then fails, and every other lookup records whether the list
// cancelled it.
type gatedInfo struct {
	core.InfoService
	gate      int
	wait      time.Duration
	fail      string
	locker    sync.Mutex
	inFlight  int
	most      int
	started   int
	cancelled int
	timedOut  int
	full      chan struct{}
	fill      sync.Once
}

func newGatedInfo(info core.InfoService, gate int, wait time.Duration, fail string) *gatedInfo {
	return &gatedInfo{InfoService: info, gate: gate, wait: wait, fail: fail, full: make(chan struct{})}
}

func (info *gatedInfo) AppExists(ctx context.Context, guid string) (bool, error) {
	info.locker.Lock()
	info.started++
	info.inFlight++
	info.most = max(info.most, info.inFlight)
	if info.inFlight == info.gate {
		info.fill.Do(func() { close(info.full) })
	}
	info.locker.Unlock()

	defer func() {
		info.locker.Lock()
		info.inFlight--
		info.locker.Unlock()
	}()

	if guid == info.fail {
		select {
		case <-info.full:
		case <-time.After(info.wait):
		}
		return false, resource.CloudFoundryHTTPError{StatusCode: http.StatusServiceUnavailable, Status: "503 Service Unavailable"}
	}

	if info.fail != "" {
		// Wait to be cancelled by the failing lookup.
		select {
		case <-ctx.Done():
			info.locker.Lock()
			info.cancelled++
			info.locker.Unlock()
			return false, ctx.Err()
		case <-time.After(info.wait):
			info.locker.Lock()
			info.timedOut++
			info.locker.Unlock()
			return info.InfoService.AppExists(ctx, guid)
		}
	}

	select {
	case <-info.full:
	case <-time.After(info.wait):
	}
	return info.InfoService.AppExists(ctx, guid)
}

// persistJobsOnApps stores one job on each of count more apps, so a list has
// that many more apps to ask about.
func (tn *tenancy) persistJobsOnApps(t *testing.T, count int) {
	t.Helper()
	for i := range count {
		app := fmt.Sprintf("app-extra-%02d", i)
		tn.cf.PrepareApp(app, "space-b")
		if _, err := tn.services.Jobs.Persist(&core.Job{Name: "job-" + app, Command: "true", AppGUID: app, SpaceGUID: "space-b"}); err != nil {
			t.Fatal(err)
		}
	}
}

// The list asks about several apps at once, but never more than
// appLookupLimit.
func TestListOrphansAsksAboutAppsConcurrently(t *testing.T) {
	tn := newTenancy(t)
	tn.persistJobsOnApps(t, 25)
	tn.cf.RemoveApp("app-b")
	info := newGatedInfo(tn.services.Info, appLookupLimit, 100*time.Millisecond, "")
	tn.services.Info = info

	orphans := tn.listOrphans(t)

	if len(orphans) != 2 {
		t.Errorf("listed %d orphans, want job B and call B", len(orphans))
	}
	if info.most < 2 || info.most > appLookupLimit {
		t.Errorf("at most %d lookups were in flight, want between 2 and %d", info.most, appLookupLimit)
	}
	if info.started != 27 {
		t.Errorf("asked about %d apps, want 27", info.started)
	}
}

// The first lookup that fails cancels the rest and fails the list.
func TestListOrphansCancelsTheOtherLookupsWhenOneFails(t *testing.T) {
	tn := newTenancy(t)
	tn.persistJobsOnApps(t, 25)
	info := newGatedInfo(tn.services.Info, appLookupLimit, 300*time.Millisecond, "app-extra-00")
	tn.services.Info = info

	rec := tn.do(http.MethodGet, "/orphans", "", tn.admin)

	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "app-extra-00") {
		t.Errorf("status %d, body %s, want %d naming app-extra-00", rec.Code, rec.Body, http.StatusBadGateway)
	}
	if info.timedOut != 0 || info.cancelled == 0 {
		t.Errorf("%d lookups ran on and %d were cancelled, want none to run on", info.timedOut, info.cancelled)
	}
	if info.started > appLookupLimit {
		t.Errorf("asked about %d apps after one failed, want at most %d", info.started, appLookupLimit)
	}
}
