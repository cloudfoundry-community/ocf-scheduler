package commands_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

const (
	unauthorized = "response status: 401 Unauthorized: the scheduler rejected the token, so refresh SCHEDULER_TOKEN with `cf oauth-token`"
	forbidden    = "response status: 403 Forbidden: the token's user is not allowed to do this"
)

// invocation is one jobs or calls command with arguments that satisfy it.
type invocation struct {
	args []string

	// looksUp is true when the command fetches the job or call by GUID
	// before it does anything else.
	looksUp bool
}

func invocations(kind string) []invocation {
	create := []string{kind, "create", "app-1", "name", "command"}
	if kind == "calls" {
		create = []string{kind, "create", "app-1", "name", "https://example.com", "auth"}
	}

	return []invocation{
		{args: []string{kind, "all", "space-1"}},
		{args: create},
		{args: []string{kind, "del", "g-1"}, looksUp: true},
		{args: []string{kind, "execute", "g-1"}, looksUp: true},
		{args: []string{kind, "history", "g-1"}, looksUp: true},
		{args: []string{kind, "schedules", "create", "g-1", "* * * * *"}, looksUp: true},
		{args: []string{kind, "schedules", "del", "g-1", "s-1"}, looksUp: true},
		{args: []string{kind, "schedules", "history", "g-1", "s-1"}, looksUp: true},
		{args: []string{kind, "schedules", "list", "g-1"}, looksUp: true},
	}
}

func run(t *testing.T, handler http.HandlerFunc, args []string) error {
	t.Helper()

	server := httptest.NewServer(handler)
	defer server.Close()

	driver, err := core.NewDriver(server.URL, "bearer token")
	if err != nil {
		t.Fatal(err)
	}
	core.Client = driver

	commands.RootCmd.SetArgs(args)

	return commands.RootCmd.Execute()
}

func refuse(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		fmt.Fprint(w, "\"\"\n")
	}
}

// lookupThenRefuse finds the job or call and refuses everything after it.
func lookupThenRefuse(kind string, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+kind+"/g-1" {
			fmt.Fprint(w, `{"guid":"g-1","name":"found"}`)
			return
		}

		refuse(status)(w, r)
	}
}

func TestCommandsReportAnEmptyRefusal(t *testing.T) {
	refusals := []struct {
		status int
		want   string
	}{
		{http.StatusUnauthorized, unauthorized},
		{http.StatusForbidden, forbidden},
	}

	for _, kind := range []string{"jobs", "calls"} {
		for _, inv := range invocations(kind) {
			for _, refusal := range refusals {
				name := fmt.Sprintf("%s/%d", strings.Join(inv.args, " "), refusal.status)

				t.Run("refused at the start "+name, func(t *testing.T) {
					err := run(t, refuse(refusal.status), inv.args)

					if err == nil || err.Error() != refusal.want {
						t.Errorf("got %v, want %q", err, refusal.want)
					}
				})

				if !inv.looksUp {
					continue
				}

				t.Run("refused after the lookup "+name, func(t *testing.T) {
					err := run(t, lookupThenRefuse(kind, refusal.status), inv.args)

					if err == nil || err.Error() != refusal.want {
						t.Errorf("got %v, want %q", err, refusal.want)
					}
				})
			}
		}
	}
}

func TestLookupsSayWhenNothingHasThatName(t *testing.T) {
	notFound := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "\"\"\n")
	}

	for _, kind := range []string{"jobs", "calls"} {
		want := "couldn't find that " + strings.TrimSuffix(kind, "s")

		for _, inv := range invocations(kind) {
			if !inv.looksUp {
				continue
			}

			t.Run(strings.Join(inv.args, " "), func(t *testing.T) {
				err := run(t, notFound, inv.args)

				if err == nil || err.Error() != want {
					t.Errorf("got %v, want %q", err, want)
				}
			})
		}
	}
}

func TestLookupsKeepTheMessageOfOtherFailures(t *testing.T) {
	broken := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}

	err := run(t, broken, []string{"jobs", "history", "g-1"})

	want := "couldn't look up that job: response status: 502 Bad Gateway"
	if err == nil || err.Error() != want {
		t.Errorf("got %v, want %q", err, want)
	}
}

// capture runs fn and returns what it wrote to standard output.
func capture(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	original := os.Stdout
	os.Stdout = writer

	runErr := fn()

	os.Stdout = original
	writer.Close()

	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	return string(out), runErr
}

func TestEmptyListsStillPrintTheirEmptyText(t *testing.T) {
	empty := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/jobs/g-1" || r.URL.Path == "/calls/g-1" {
			fmt.Fprint(w, `{"guid":"g-1","name":"found"}`)
			return
		}

		fmt.Fprint(w, `{"resources":[]}`)
	}

	for _, kind := range []string{"jobs", "calls"} {
		tests := []struct {
			args []string
			want string
		}{
			{[]string{kind, "history", "g-1"}, "No executions for " + strings.TrimSuffix(kind, "s") + " g-1"},
			{[]string{kind, "schedules", "history", "g-1", "s-1"}, "No executions for schedule s-1"},
			{[]string{kind, "schedules", "list", "g-1"}, "No schedules for " + strings.TrimSuffix(kind, "s") + " g-1"},
		}

		for _, tt := range tests {
			t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
				out, err := capture(t, func() error { return run(t, empty, tt.args) })

				if err != nil {
					t.Errorf("unexpected error %v", err)
				}
				if !strings.Contains(out, tt.want) {
					t.Errorf("output %q lacks %q", out, tt.want)
				}
			})
		}
	}
}

// actions are the commands that change something after the lookup.
func actions(kind string) []invocation {
	var found []invocation

	for _, inv := range invocations(kind) {
		switch inv.args[1] {
		case "del", "execute":
			found = append(found, inv)
		case "schedules":
			if inv.args[2] == "create" || inv.args[2] == "del" {
				found = append(found, inv)
			}
		}
	}

	return found
}

func TestActionsKeepTheMessageOfOtherFailures(t *testing.T) {
	lookup := func(kind string) func(w http.ResponseWriter, r *http.Request) bool {
		return func(w http.ResponseWriter, r *http.Request) bool {
			if r.Method == http.MethodGet && r.URL.Path == "/"+kind+"/g-1" {
				fmt.Fprint(w, `{"guid":"g-1","name":"found"}`)
				return true
			}

			return false
		}
	}

	for _, kind := range []string{"jobs", "calls"} {
		for _, inv := range actions(kind) {
			t.Run("5xx "+strings.Join(inv.args, " "), func(t *testing.T) {
				find := lookup(kind)
				err := run(t, func(w http.ResponseWriter, r *http.Request) {
					if !find(w, r) {
						w.WriteHeader(http.StatusServiceUnavailable)
					}
				}, inv.args)

				if err == nil || !strings.HasPrefix(err.Error(), "couldn't ") ||
					!strings.HasSuffix(err.Error(), ": response status: 503 Service Unavailable") {
					t.Errorf("unexpected error %v", err)
				}
			})

			t.Run("network "+strings.Join(inv.args, " "), func(t *testing.T) {
				find := lookup(kind)
				err := run(t, func(w http.ResponseWriter, r *http.Request) {
					if find(w, r) {
						return
					}

					conn, _, hijackErr := w.(http.Hijacker).Hijack()
					if hijackErr != nil {
						t.Error(hijackErr)
						return
					}
					conn.Close()
				}, inv.args)

				if err == nil || !strings.HasPrefix(err.Error(), "couldn't ") ||
					strings.Contains(err.Error(), "response status") {
					t.Errorf("unexpected error %v", err)
				}
			})
		}
	}
}

func TestLookupsKeepTheDecodeError(t *testing.T) {
	garbled := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not json")
	}

	for _, kind := range []string{"jobs", "calls"} {
		prefix := "couldn't look up that " + strings.TrimSuffix(kind, "s") + ": invalid character"

		for _, inv := range invocations(kind) {
			if !inv.looksUp {
				continue
			}

			t.Run(strings.Join(inv.args, " "), func(t *testing.T) {
				err := run(t, garbled, inv.args)

				if err == nil || !strings.HasPrefix(err.Error(), prefix) {
					t.Errorf("got %v, want prefix %q", err, prefix)
				}
			})
		}
	}
}
