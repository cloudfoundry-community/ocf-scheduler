package cf_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	cfconfig "github.com/cloudfoundry/go-cfclient/v3/config"

	"github.com/cloudfoundry-community/ocf-scheduler/cf"
)

// fakeFoundation answers UAA token requests and app lookups in process. It
// records the credentials each token request sends and the bearer token
// each app lookup carries.
type fakeFoundation struct {
	lock          sync.Mutex
	issued        int
	tokenRequests []string
	appBearers    []string
}

func (f *fakeFoundation) RoundTrip(req *http.Request) (*http.Response, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	switch {
	case req.URL.Path == "/oauth/token":
		user, _, _ := req.BasicAuth()
		body, _ := io.ReadAll(req.Body)
		f.tokenRequests = append(f.tokenRequests, user+" "+string(body))
		f.issued++
		payload, _ := json.Marshal(map[string]any{"scope": []string{"cloud_controller.admin_read_only"}, "jti": fmt.Sprint(f.issued)})
		token := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
		return respond(`{"access_token":"` + token + `","token_type":"bearer","expires_in":3600}`), nil
	case strings.HasPrefix(req.URL.Path, "/v3/apps/"):
		f.appBearers = append(f.appBearers, req.Header.Get("Authorization"))
		return respond(`{"guid":"app-1"}`), nil
	}

	return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}, Request: req}, nil
}

func respond(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

// Token asks UAA with the credentials and requested scopes that the client's
// Cloud Controller calls use, and reuses the token it gets.
func TestRealCFClient_TokenUsesTheClientsOwnCredentials(t *testing.T) {
	foundation := &fakeFoundation{}
	cfg, err := cfconfig.New("https://api.example.com",
		cfconfig.ClientCredentials("scheduler", "secret"),
		cfconfig.AuthTokenURL("https://login.example.com", "https://uaa.example.com"),
		cfconfig.HttpClient(&http.Client{Transport: foundation}),
	)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := cfclient.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	client := cf.NewRealCFClient(raw)

	if _, err := client.GetApp(t.Context(), "app-1"); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		token, err := client.Token()
		if err != nil {
			t.Fatal(err)
		}
		claims, err := cf.DecodeJWT(token.AccessToken)
		if err != nil {
			t.Fatal(err)
		}
		if len(claims.Scope) != 1 || claims.Scope[0] != "cloud_controller.admin_read_only" {
			t.Errorf("scopes %v, want cloud_controller.admin_read_only", claims.Scope)
		}
	}

	if len(foundation.appBearers) != 1 || !strings.HasPrefix(foundation.appBearers[0], "Bearer ") {
		t.Fatalf("app lookups carried %v, want one bearer token", foundation.appBearers)
	}
	if len(foundation.tokenRequests) != 2 {
		t.Fatalf("UAA got %d token requests, want one for the app lookup and one reused for both Token calls: %v", len(foundation.tokenRequests), foundation.tokenRequests)
	}
	if foundation.tokenRequests[0] != foundation.tokenRequests[1] {
		t.Errorf("Token asked UAA with %q, but the app lookup asked with %q", foundation.tokenRequests[1], foundation.tokenRequests[0])
	}
	if !strings.HasPrefix(foundation.tokenRequests[0], "scheduler ") {
		t.Errorf("token request %q does not use the scheduler's client ID", foundation.tokenRequests[0])
	}
}
