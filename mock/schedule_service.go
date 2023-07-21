package mock

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

type ScheduleService struct {
	storage []*core.Schedule
	locker  sync.Mutex
}

func NewScheduleService() *ScheduleService {
	return &ScheduleService{
		storage: make([]*core.Schedule, 0),
	}
}

func (service *ScheduleService) Persist(candidate *core.Schedule) (*core.Schedule, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	now := time.Now().UTC()

	id, err := uuid.NewRandom()

	if err != nil {
		return nil, fmt.Errorf("could not generate a job id")
	}

	candidate.GUID = id.String()
	candidate.CreatedAt = now
	candidate.UpdatedAt = now

	service.storage = append(service.storage, candidate)

	return candidate, nil
}

func (service *ScheduleService) ByCall(call *core.Call) []*core.Schedule {
	service.locker.Lock()
	defer service.locker.Unlock()

	found := make([]*core.Schedule, 0)

	for _, candidate := range service.storage {
		if candidate.RefType == "call" && candidate.RefGUID == call.GUID {
			found = append(found, candidate)
		}
	}

	return found
}

func (service *ScheduleService) ByJob(job *core.Job) []*core.Schedule {
	service.locker.Lock()
	defer service.locker.Unlock()

	found := make([]*core.Schedule, 0)

	for _, candidate := range service.storage {
		if candidate.RefType == "job" && candidate.RefGUID == job.GUID {
			found = append(found, candidate)
		}
	}

	return found
}

func (service *ScheduleService) Get(guid string) (*core.Schedule, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	candidates := make([]*core.Schedule, 0)

	for _, schedule := range service.storage {
		if schedule.GUID == guid {
			candidates = append(candidates, schedule)
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

func (service *ScheduleService) Delete(schedule *core.Schedule) error {
	service.locker.Lock()
	defer service.locker.Unlock()

	keep := make([]*core.Schedule, 0)
	for _, item := range service.storage {
		if item.GUID != schedule.GUID {
			keep = append(keep, item)
		}
	}

	service.storage = keep

	return nil
}

func (service *ScheduleService) Enabled() []*core.Schedule {
	service.locker.Lock()
	defer service.locker.Unlock()

	enabled := make([]*core.Schedule, 0)

	for _, candidate := range service.storage {
		if candidate.Enabled {
			enabled = append(enabled, candidate)
		}
	}

	return enabled
}
