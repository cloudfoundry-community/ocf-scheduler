package list

import (
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
