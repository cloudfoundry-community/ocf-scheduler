package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Most commands are open to space members, so their 401 and 403 say nothing
// about admins.
func TestResultErrorExplainsAnEmptyBody(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   string
	}{
		{
			name:   "a rejected token",
			status: 401,
			want:   "response status: 401 Unauthorized: the scheduler rejected the token, so refresh SCHEDULER_TOKEN with `cf oauth-token`",
		},
		{
			name:   "a user who may not do this",
			status: 403,
			want:   "response status: 403 Forbidden: the token's user is not allowed to do this",
		},
		{
			name:   "a gateway failure",
			status: 502,
			want:   "response status: 502 Bad Gateway",
		},
		{
			name:   "a status with no name",
			status: 599,
			want:   "response status: 599",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{Status: tt.status}

			if got := result.Error().Error(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// Only an admin may run the orphans commands, so their 401 and 403 say so.
func TestResultAdminErrorExplainsAnEmptyBody(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{401, "response status: 401 Unauthorized: the scheduler rejected the token, so refresh SCHEDULER_TOKEN with `cf oauth-token` as an admin user"},
		{403, "response status: 403 Forbidden: the token's user is not an admin"},
		{502, "response status: 502 Bad Gateway"},
	}

	for _, tt := range tests {
		result := &Result{Status: tt.status}

		if got := result.AdminError().Error(); got != tt.want {
			t.Errorf("%d: got %q, want %q", tt.status, got, tt.want)
		}
	}
}

// The scheduler's own message wins over the generic explanation for the
// status, because it says more.
func TestResultErrorPrefersTheBodysMessage(t *testing.T) {
	result := &Result{Status: 401, Body: []byte(`"token expired"`)}

	if got, want := result.Error().Error(), "response status: 401: token expired"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResultErrorBoundsAPlainTextBody(t *testing.T) {
	result := &Result{Status: 502, Body: []byte(strings.Repeat("x", 2*maxMessage))}

	if got, want := result.Error().Error(), "response status: 502: "+strings.Repeat("x", maxMessage)+"..."; got != want {
		t.Errorf("got %d characters, want %d", len(got), len(want))
	}
}

// The scheduler refuses a request with the JSON empty string and a newline,
// and a gateway may send nothing or whitespace, so all of them are empty.
func TestMessageTreatsABlankBodyAsEmpty(t *testing.T) {
	for _, body := range []string{"", "\n", "  \n", `""`, "\"\"\n", `"  "`} {
		if got := Message([]byte(body)); got != "" {
			t.Errorf("Message(%q) = %q, want empty", body, got)
		}
	}
}

// Get, Post, and Delete keep the 20 second wait that they always had, and
// only Fetch waits as long as an orphan list needs.
func TestRequestTimeouts(t *testing.T) {
	if requestClient.Timeout != 20*time.Second {
		t.Errorf("requestClient waits %s, want 20s", requestClient.Timeout)
	}
	if fetchClient.Timeout != 15*time.Minute {
		t.Errorf("fetchClient waits %s, want 15m", fetchClient.Timeout)
	}
}

func TestSendUsesTheRequestClient(t *testing.T) {
	saved := requestClient
	defer func() { requestClient = saved }()
	requestClient = &http.Client{Timeout: 50 * time.Millisecond}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	driver, err := NewDriver(server.URL, "bearer token")
	if err != nil {
		t.Fatal(err)
	}

	for name, result := range map[string]*Result{
		"get":    driver.Get("jobs", nil),
		"post":   driver.Post("jobs", nil, nil),
		"delete": driver.Delete("jobs/1", nil),
	} {
		if result.Okay() || result.Error() == nil {
			t.Errorf("%s did not time out on the request client", name)
		}
	}
}
