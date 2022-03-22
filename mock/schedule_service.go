package mock

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

type ScheduleService struct {
	storage []*core.Schedule
}

func NewScheduleService() *ScheduleService {
	return &ScheduleService{
		storage: make([]*core.Schedule, 0),
	}
}

func (service *ScheduleService) Persist(candidate *core.Schedule) (*core.Schedule, error) {
	now := time.Now().UTC()

	id, err := uuid.NewRandom()

	if err != nil {
		return nil, fmt.Errorf("could not generate a job id")
	}

	candidate.GUID = id.String()
	candidate.CreatedAt = now.String()
	candidate.UpdatedAt = now.String()

	service.storage = append(service.storage, candidate)

	return candidate, nil
}

func (service *ScheduleService) ByJob(job *core.Job) []*core.Schedule {
	found := make([]*core.Schedule, 0)

	for _, candidate := range service.storage {
		if candidate.RefType == "job" && candidate.RefGUID == job.GUID {
			found = append(found, candidate)
		}
	}

	return found
}
