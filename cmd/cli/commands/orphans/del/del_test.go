package del

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

func run(t *testing.T, status int, body string) error {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.URL.Path != "/orphans/delete" || !strings.Contains(string(sent), "guid-1") {
			t.Errorf("unexpected request %s %s with body %s", r.Method, r.URL.Path, sent)
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

	return Command.RunE(Command, []string{"guid-1"})
}

func TestDelShowsTheServersMessageOnA500(t *testing.T) {
	err := run(t, http.StatusInternalServerError, `"the scheduler's Cloud Controller client needs the cloud_controller.admin scope"`)

	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "needs the cloud_controller.admin scope") {
		t.Errorf("error %q does not carry the status and the message", err)
	}
}

func TestDelShowsThe502Message(t *testing.T) {
	err := run(t, http.StatusBadGateway, `"could not read the scopes on the scheduler's Cloud Controller token"`)

	if err == nil || !strings.Contains(err.Error(), "502") || !strings.Contains(err.Error(), "could not read the scopes") {
		t.Errorf("error %v does not carry the status and the message", err)
	}
}

func TestDelSucceedsWhenEverythingIsDeleted(t *testing.T) {
	err := run(t, http.StatusOK, `{"results":[{"guid":"guid-1","type":"job","result":"deleted"}]}`)

	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestDelFailsWhenAGUIDIsKept(t *testing.T) {
	err := run(t, http.StatusOK, `{"results":[{"guid":"guid-1","type":"job","result":"app_exists","message":"app a exists"}]}`)

	if err == nil || err.Error() != "1 of 1 were not deleted" {
		t.Errorf("unexpected error %v", err)
	}
}

func TestDelShowsARefusedSchedule(t *testing.T) {
	var printed bytes.Buffer
	Command.SetOut(&printed)
	defer Command.SetOut(nil)

	err := run(t, http.StatusOK, `{"results":[{"guid":"guid-1","type":"schedule","result":"parent_exists","message":"job job-1 exists"}]}`)

	if err == nil || err.Error() != "1 of 1 were not deleted" {
		t.Errorf("unexpected error %v", err)
	}
	if !strings.Contains(printed.String(), "parent_exists") || !strings.Contains(printed.String(), "job job-1 exists") {
		t.Errorf("the output does not show the refusal:\n%s", printed.String())
	}
}
