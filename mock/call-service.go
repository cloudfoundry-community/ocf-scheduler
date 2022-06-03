package mock

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/starkandwayne/scheduler-for-ocf/core"
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

	// this is just the easiest way to force a failure in the mock service
	if call.Name == "sad-face" {
		return fmt.Errorf("such a sad thing")
	}

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

func (service *CallService) Exists(appguid string, name string) bool {
	service.locker.Lock()
	defer service.locker.Unlock()

	return service.exists(appguid, name)
}

func (service *CallService) exists(appguid string, name string) bool {
	candidates := make([]*core.Call, 0)

	for _, call := range service.storage {
		if call.Name == name && call.AppGUID == appguid {
			candidates = append(candidates, call)
		}
	}

	return len(candidates) > 0
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

func (service *CallService) Reset() {
	service.locker.Lock()
	defer service.locker.Unlock()

	service.storage = make([]*core.Call, 0)
}

func NewCallService() *CallService {
	service := &CallService{}
	service.Reset()

	return service
}
