package mock

var SpaceGUID = "abcdef-1"

type InfoService struct{}

func (service *InfoService) GetSpaceGUIDForApp(guid string) (string, error) {
	return SpaceGUID, nil
}

func NewInfoService() *InfoService {
	return &InfoService{}
}
