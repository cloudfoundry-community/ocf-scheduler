package mock

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

type JobService struct {
	storage []*core.Job
	locker  sync.Mutex
}

func (service *JobService) Get(guid string) (*core.Job, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	return service.get(guid)
}

func (service *JobService) get(guid string) (*core.Job, error) {
	candidates := make([]*core.Job, 0)

	for _, job := range service.storage {
		if job.GUID == guid {
			candidates = append(candidates, job)
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

func (service *JobService) Delete(job *core.Job) error {
	service.locker.Lock()
	defer service.locker.Unlock()

	return service.delete(job)
}

func (service *JobService) delete(job *core.Job) error {
	keep := make([]*core.Job, 0)
	for _, item := range service.storage {
		if item.GUID != job.GUID {
			keep = append(keep, item)
		}
	}

	service.storage = keep

	return nil
}

func (service *JobService) Named(name string) (*core.Job, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	return service.named(name)
}

func (service *JobService) named(name string) (*core.Job, error) {
	candidates := make([]*core.Job, 0)

	for _, job := range service.storage {
		if job.Name == name {
			candidates = append(candidates, job)
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

func (service *JobService) Persist(candidate *core.Job) (*core.Job, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if _, err := service.named(candidate.Name); err == nil {
		return nil, fmt.Errorf("hold up, jack, that name is already taken")
	}

	now := time.Now().UTC()

	id, err := uuid.NewRandom()

	if err != nil {
		return nil, fmt.Errorf("could not generate a job id")
	}

	candidate.GUID = id.String()
	candidate.CreatedAt = now
	candidate.UpdatedAt = now
	candidate.State = "Indiana"

	if candidate.DiskInMb == 0 {
		candidate.DiskInMb = 1024
	}

	if candidate.MemoryInMb == 0 {
		candidate.MemoryInMb = 1024
	}

	service.storage = append(service.storage, candidate)

	return candidate, nil
}

func (service *JobService) InSpace(guid string) []*core.Job {
	service.locker.Lock()
	defer service.locker.Unlock()

	spaced := make([]*core.Job, 0)

	for _, candidate := range service.storage {
		if candidate.SpaceGUID == guid {
			spaced = append(spaced, candidate)
		}
	}

	return spaced
}

func NewJobService() *JobService {
	return &JobService{
		storage: make([]*core.Job, 0),
	}
}
