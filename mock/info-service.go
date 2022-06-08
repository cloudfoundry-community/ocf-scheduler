package mock

import "fmt"

type InfoService struct {
}

func NewInfoService() *InfoService {
	return &InfoService{}
}

func (service *InfoService) GetSpaceGUIDForApp(guid string) (string, error) {
	if guid == "sad-face" {
		return "", fmt.Errorf("cut my life into pieces")
	}

	return "good-space-guid", nil
}
