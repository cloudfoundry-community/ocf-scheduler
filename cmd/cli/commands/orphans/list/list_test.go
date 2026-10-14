package list

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

func run(t *testing.T, status int, body string) error {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orphans" || r.Header.Get("Authorization") != "bearer token" {
			t.Errorf("unexpected request %s with auth %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	defer server.Close()

	driver, err := core.NewDriver(server.URL, "bearer token")
	if err != nil {
		t.Fatal(err)
	}
	core.Client = driver

	return Command.RunE(Command, nil)
}

func TestListShowsTheServersMessageOnA500(t *testing.T) {
	err := run(t, http.StatusInternalServerError, `"the scheduler's Cloud Controller client needs the cloud_controller.admin scope"`)

	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "needs the cloud_controller.admin scope") {
		t.Errorf("error %q does not carry the status and the message", err)
	}
}

func TestListShowsThe502Message(t *testing.T) {
	err := run(t, http.StatusBadGateway, `"could not read the scopes on the scheduler's Cloud Controller token"`)

	if err == nil || !strings.Contains(err.Error(), "502") || !strings.Contains(err.Error(), "could not read the scopes") {
		t.Errorf("error %v does not carry the status and the message", err)
	}
}

func TestListShowsAGatewaysPlainTextBody(t *testing.T) {
	err := run(t, http.StatusBadGateway, "Bad Gateway\n")

	if err == nil || err.Error() != "response status: 502: Bad Gateway" {
		t.Errorf("unexpected error %v", err)
	}
}

func TestListWithAnEmptyBodyShowsTheStatus(t *testing.T) {
	err := run(t, http.StatusForbidden, "")

	if err == nil || err.Error() != "response status: 403 Forbidden: the token's user is not an admin" {
		t.Errorf("unexpected error %v", err)
	}
}

func TestListSucceeds(t *testing.T) {
	if err := run(t, http.StatusOK, `{"resources":[]}`); err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

// output runs the command and returns what it printed.
func output(t *testing.T, status int, body string) (string, error) {
	t.Helper()

	var printed bytes.Buffer
	Command.SetOut(&printed)
	defer Command.SetOut(nil)

	err := run(t, status, body)
	return printed.String(), err
}

func TestListShowsSchedulesWhoseOwnerIsGone(t *testing.T) {
	printed, err := output(t, http.StatusOK, `{
		"resources": [],
		"orphan_schedules": [
			{"guid": "sched-1", "enabled": true, "expression": "0 2 * * *", "expression_type": "cron_expression", "ref_type": "job", "ref_guid": "job-1"},
			{"guid": "sched-2", "enabled": false, "expression": "0 3 * * *", "expression_type": "cron_expression", "ref_type": "call", "ref_guid": "call-1"}
		]
	}`)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if !strings.Contains(printed, "No jobs or calls whose app is gone") {
		t.Errorf("the output does not say no jobs or calls are orphaned:\n%s", printed)
	}
	for _, line := range [][]string{
		{"sched-1", "job", "job-1", "0 2 * * *", "true"},
		{"sched-2", "call", "call-1", "0 3 * * *", "false"},
	} {
		if !lineWith(printed, line...) {
			t.Errorf("no line shows %v:\n%s", line, printed)
		}
	}
}

func TestListSaysWhenNoScheduleIsOrphaned(t *testing.T) {
	printed, err := output(t, http.StatusOK, `{"resources": [], "orphan_schedules": []}`)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if !strings.Contains(printed, "No schedules whose job or call is gone") {
		t.Errorf("the output does not say no schedules are orphaned:\n%s", printed)
	}
}

// A scheduler too old to list orphan schedules leaves the field out, and the
// CLI must not claim there are none.
func TestListSaysNothingAboutSchedulesAnOlderSchedulerLeftOut(t *testing.T) {
	printed, err := output(t, http.StatusOK, `{"resources": []}`)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if strings.Contains(printed, "schedules whose job or call is gone") {
		t.Errorf("the output speaks for schedules the scheduler did not list:\n%s", printed)
	}
}

// lineWith reports whether one line of printed holds every field.
func lineWith(printed string, fields ...string) bool {
	for _, line := range strings.Split(printed, "\n") {
		all := true
		for _, field := range fields {
			if !strings.Contains(line, field) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}

	return false
}
