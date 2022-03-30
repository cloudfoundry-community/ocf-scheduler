package cf

type InfoService struct {
	client Client
}

func NewInfoService(client Client) *InfoService {
	return &InfoService{client: client}
}

func (service *InfoService) GetSpaceGUIDForApp(guid string) (string, error) {
	app, err := service.client.AppByGuid(guid)
	if err != nil {
		return "", err
	}

	return app.SpaceGuid, nil
}
