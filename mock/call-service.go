package mock

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

type CallService struct {
	storage []*core.Call
	locker  sync.Mutex
}

func (service *CallService) Get(guid string) (*core.Call, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	return service.get(guid)
}

func (service *CallService) get(guid string) (*core.Call, error) {
	candidates := make([]*core.Call, 0)

	for _, call := range service.storage {
		if call.GUID == guid {
			candidates = append(candidates, call)
		}
	}

	if len(candidates) > 1 {
		return nil, fmt.Errorf("too many results")
	}

	if len(candidates) < 1 {
		return nil, fmt.Errorf("too few results")
	}

	return candidates[0], nil
}

func (service *CallService) Delete(call *core.Call) error {
	service.locker.Lock()
	defer service.locker.Unlock()

	keep := make([]*core.Call, 0)
	for _, item := range service.storage {
		if item.GUID != call.GUID {
			keep = append(keep, item)
		}
	}

	service.storage = keep

	return nil
}

func (service *CallService) Named(name string) (*core.Call, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	return service.named(name)
}

func (service *CallService) named(name string) (*core.Call, error) {
	candidates := make([]*core.Call, 0)

	for _, call := range service.storage {
		if call.Name == name {
			candidates = append(candidates, call)
		}
	}

	if len(candidates) > 1 {
		return nil, fmt.Errorf("too many results")
	}

	if len(candidates) < 1 {
		return nil, fmt.Errorf("too few results")
	}

	return candidates[0], nil
}

func (service *CallService) Persist(candidate *core.Call) (*core.Call, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if _, err := service.named(candidate.Name); err == nil {
		return nil, fmt.Errorf("hold up, jack, that name is already taken")
	}

	now := time.Now().UTC()

	id, err := uuid.NewRandom()

	if err != nil {
		return nil, fmt.Errorf("could not generate a call id")
	}

	candidate.GUID = id.String()
	candidate.CreatedAt = now
	candidate.UpdatedAt = now

	service.storage = append(service.storage, candidate)

	return candidate, nil
}

func (service *CallService) InSpace(guid string) []*core.Call {
	service.locker.Lock()
	defer service.locker.Unlock()

	spaced := make([]*core.Call, 0)

	for _, candidate := range service.storage {
		if candidate.SpaceGUID == guid {
			spaced = append(spaced, candidate)
		}
	}

	return spaced
}

func NewCallService() *CallService {
	return &CallService{
		storage: make([]*core.Call, 0),
	}
}
