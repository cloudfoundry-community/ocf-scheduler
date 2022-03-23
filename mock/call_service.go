package mock

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

type CallService struct {
	storage []*core.Call
}

func (service *CallService) Get(guid string) (*core.Call, error) {
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
	if _, err := service.Named(candidate.Name); err == nil {
		return nil, fmt.Errorf("hold up, jack, that name is already taken")
	}

	now := time.Now().UTC()

	id, err := uuid.NewRandom()

	if err != nil {
		return nil, fmt.Errorf("could not generate a call id")
	}

	candidate.GUID = id.String()
	candidate.CreatedAt = now.String()
	candidate.UpdatedAt = now.String()

	service.storage = append(service.storage, candidate)

	return candidate, nil
}

func (service *CallService) InSpace(guid string) []*core.Call {
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
