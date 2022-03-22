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
