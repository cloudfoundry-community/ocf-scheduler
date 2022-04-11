package cf

import (
	cf "github.com/cloudfoundry-community/go-cfclient"
)

type InfoService struct {
	client *cf.Client
}

func NewInfoService(client *cf.Client) *InfoService {
	return &InfoService{client: client}
}

func (service *InfoService) GetSpaceGUIDForApp(guid string) (string, error) {
	app, err := service.client.AppByGuid(guid)
	if err != nil {
		return "", err
	}

	return app.SpaceGuid, nil
}
