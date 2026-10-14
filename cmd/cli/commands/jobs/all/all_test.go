package all

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

// run answers the command with the status and body, as the scheduler does
// for a refusal.
func run(t *testing.T, status int, body string) error {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jobs" || r.URL.Query().Get("space_guid") != "space-1" {
			t.Errorf("unexpected request %s", r.URL)
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

	return Command.RunE(Command, []string{"space-1"})
}

func TestAllExplainsAnEmptyRefusal(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{http.StatusUnauthorized, "response status: 401 Unauthorized: the scheduler rejected the token, so refresh SCHEDULER_TOKEN with `cf oauth-token`"},
		{http.StatusForbidden, "response status: 403 Forbidden: the token's user is not allowed to do this"},
	}

	for _, tt := range tests {
		err := run(t, tt.status, "\"\"\n")

		if err == nil || err.Error() != tt.want {
			t.Errorf("%d: got %v, want %q", tt.status, err, tt.want)
		}
	}
}

func TestAllShowsTheServersMessage(t *testing.T) {
	err := run(t, http.StatusForbidden, `"not your space"`)

	if err == nil || err.Error() != "response status: 403: not your space" {
		t.Errorf("unexpected error %v", err)
	}
}
