package cf

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/go-cfclient/v3/resource"
)

type InfoService struct {
	client CFClient
}

func NewInfoService(client CFClient) *InfoService {
	return &InfoService{client: client}
}

func (service *InfoService) GetSpaceGUIDForApp(guid string) (string, error) {
	app, err := service.client.GetApp(context.Background(), guid)
	if err != nil {
		return "", err
	}

	if app.Relationships.Space.Data == nil {
		return "", fmt.Errorf("app has no space relationship")
	}

	return app.Relationships.Space.Data.GUID, nil
}

// ServiceInstanceExists answers false only when Cloud Controller reports the
// instance as not found. Any other failure is an error, so a caller never
// mistakes an unreachable API for a purged instance.
func (service *InfoService) ServiceInstanceExists(ctx context.Context, guid string) (bool, error) {
	_, err := service.client.GetServiceInstance(ctx, guid)
	if err == nil {
		return true, nil
	}

	if resource.IsResourceNotFoundError(err) || resource.IsServiceInstanceNotFoundError(err) {
		return false, nil
	}

	return false, err
}
