package mock

import (
	"sync"
)

type LogService struct {
	infos      map[string][]string
	errs       map[string][]string
	infoLocker sync.Mutex
	errLocker  sync.Mutex
}

func (service *LogService) Info(scope, message string) {
	service.infoLocker.Lock()
	defer service.infoLocker.Unlock()

	if service.infos == nil {
		service.infos = make(map[string][]string)
	}

	if service.infos[scope] == nil {
		service.infos[scope] = make([]string, 0)
	}

	service.infos[scope] = append(service.infos[scope], message)
}

func (service *LogService) Error(scope, message string) {
	service.errLocker.Lock()
	defer service.errLocker.Unlock()

	if service.errs == nil {
		service.errs = make(map[string][]string)
	}

	if service.errs[scope] == nil {
		service.errs[scope] = make([]string, 0)
	}

	service.errs[scope] = append(service.errs[scope], message)
}

func (service *LogService) ReceivedInfo(scope, message string) bool {
	service.infoLocker.Lock()
	defer service.infoLocker.Unlock()

	for _, line := range service.infos[scope] {
		if line == message {
			return true
		}
	}

	return false
}

func (service *LogService) ReceivedError(scope, message string) bool {
	service.errLocker.Lock()
	defer service.errLocker.Unlock()

	for _, line := range service.errs[scope] {
		if line == message {
			return true
		}
	}

	return false
}

func (service *LogService) ErrorsFor(scope string) []string {
	service.errLocker.Lock()
	defer service.errLocker.Unlock()

	return service.errs[scope]
}

func (service *LogService) Reset() {
	service.errs = make(map[string][]string)
	service.infos = make(map[string][]string)
}

func NewLogService() *LogService {
	return &LogService{}
}
