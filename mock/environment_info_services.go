package mock

var SpaceGUID = "abcdef-1"

type EnvironmentInfoService struct{}

func (service *EnvironmentInfoService) SpaceGUID() string {
	return SpaceGUID
}

func NewEnvironmentInfoService() *EnvironmentInfoService {
	return &EnvironmentInfoService{}
}
