package cf

import (
	"context"
	"fmt"
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
