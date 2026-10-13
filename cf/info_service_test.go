package cf_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/cloudfoundry/go-cfclient/v3/resource"

	"github.com/cloudfoundry-community/ocf-scheduler/cf"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

func TestInfoService_AppExists(t *testing.T) {
	client, err := mock.NewCFClient()
	if err != nil {
		t.Fatal(err)
	}
	info := cf.NewInfoService(client)

	client.PrepareApp("app-1", "space-1")

	exists, err := info.AppExists(context.Background(), "app-1")
	if err != nil || !exists {
		t.Errorf("a known app: exists %t, error %v, want true and no error", exists, err)
	}

	client.RemoveApp("app-1")

	exists, err = info.AppExists(context.Background(), "app-1")
	if err != nil || exists {
		t.Errorf("a deleted app: exists %t, error %v, want false and no error", exists, err)
	}
}

// Only Cloud Controller's own not-found answer counts as a deleted app. A
// 403, a 5xx, a bare 404 that carries no Cloud Controller error, such as a
// router answers for a missing route, or any other failure is an error.
func TestInfoService_AppExistsReportsEveryOtherFailure(t *testing.T) {
	for name, failure := range map[string]error{
		"not authorized": resource.NewNotAuthorizedError(),
		"server error":   resource.CloudFoundryHTTPError{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway"},
		"bare 404":       resource.CloudFoundryHTTPError{StatusCode: http.StatusNotFound, Status: "404 Not Found"},
		"timeout":        context.DeadlineExceeded,
		"anything else":  errors.New("connection refused"),
	} {
		t.Run(name, func(t *testing.T) {
			client, err := mock.NewCFClient()
			if err != nil {
				t.Fatal(err)
			}
			client.FailAppLookup("app-1", failure)

			exists, err := cf.NewInfoService(client).AppExists(context.Background(), "app-1")
			if err == nil {
				t.Errorf("exists %t and no error, want an error", exists)
			}
			if exists {
				t.Errorf("the app is reported to exist")
			}
		})
	}
}

func TestInfoService_ClientScopes(t *testing.T) {
	client, err := mock.NewCFClient()
	if err != nil {
		t.Fatal(err)
	}
	info := cf.NewInfoService(client)

	client.SetClientScopes("cloud_controller.global_auditor", "uaa.resource")
	scopes, err := info.ClientScopes()
	if err != nil {
		t.Fatalf("error %v, want none", err)
	}
	if !slices.Equal(scopes, []string{"cloud_controller.global_auditor", "uaa.resource"}) {
		t.Errorf("scopes %v, want cloud_controller.global_auditor and uaa.resource", scopes)
	}

	client.SetClientScopes()
	scopes, err = info.ClientScopes()
	if err != nil || len(scopes) != 0 {
		t.Errorf("a token with no scopes: scopes %v, error %v, want none and no error", scopes, err)
	}
}

// A token that cannot be had or decoded is an error, never an empty list of
// scopes.
func TestInfoService_ClientScopesReportsAnUnreadableToken(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	for name, spoil := range map[string]func(client *mock.CFClient){
		"UAA unreachable": func(client *mock.CFClient) { client.FailClientToken(errors.New("connection refused")) },
		"empty token":     func(client *mock.CFClient) { client.SetClientToken("") },
		"not a JWT":       func(client *mock.CFClient) { client.SetClientToken("opaque-token") },
		"payload not b64": func(client *mock.CFClient) { client.SetClientToken(header + ".!!!.sig") },
		"payload not JSON": func(client *mock.CFClient) {
			client.SetClientToken(header + "." + base64.RawURLEncoding.EncodeToString([]byte("nope")) + ".sig")
		},
	} {
		t.Run(name, func(t *testing.T) {
			client, err := mock.NewCFClient()
			if err != nil {
				t.Fatal(err)
			}
			spoil(client)

			scopes, err := cf.NewInfoService(client).ClientScopes()
			if err == nil {
				t.Errorf("scopes %v and no error, want an error", scopes)
			}
		})
	}
}
