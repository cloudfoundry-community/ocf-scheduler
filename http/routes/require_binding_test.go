package routes

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

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
