package cf_test

import (
	"context"
	"errors"
	"net/http"
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
