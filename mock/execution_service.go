package mock

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

type ExecutionService struct {
	storage map[string]*core.Execution
}

func NewExecutionService() *ExecutionService {
	return &ExecutionService{
		storage: make(map[string]*core.Execution),
	}
}

func (service *ExecutionService) Persist(candidate *core.Execution) (*core.Execution, error) {
	id, err := uuid.NewRandom()

	if err != nil {
		return nil, fmt.Errorf("could not generate a job id")
	}

	candidate.GUID = id.String()

	return service.update(candidate)
}

func (service *ExecutionService) update(candidate *core.Execution) (*core.Execution, error) {
	service.storage[candidate.GUID] = candidate

	return candidate, nil
}

func (service *ExecutionService) UpdateMessage(execution *core.Execution, message string) (*core.Execution, error) {
	execution.Message = message

	return service.update(execution)
}

func (service *ExecutionService) Start(execution *core.Execution) (*core.Execution, error) {
	now := time.Now().UTC()

	execution.ExecutionStartTime = now
	if len(execution.ScheduleGUID) > 0 {
		execution.ScheduledTime = time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			now.Hour(),
			now.Minute(),
			0,
			0,
			time.UTC,
		)
	}

	return service.update(execution)
}

func (service *ExecutionService) finish(execution *core.Execution, state string) (*core.Execution, error) {
	execution.ExecutionEndTime = time.Now().UTC()
	execution.State = state

	return service.update(execution)
}

func (service *ExecutionService) Success(execution *core.Execution) (*core.Execution, error) {
	return service.finish(execution, "SUCCEEDED")
}

func (service *ExecutionService) Fail(execution *core.Execution) (*core.Execution, error) {
	return service.finish(execution, "FAILED")
}

//func (service *ExecutionService) Get(guid string) (*core.Execution, error) {
//candidate, found := service.storage[guid]
//if !found {
//return nil, fmt.Errorf("not found")
//}

//return candidate, nil
//}

func (service *ExecutionService) ByJob(job *core.Job) []*core.Execution {
	found := make([]*core.Execution, 0)

	for _, candidate := range service.storage {
		if candidate.RefType == "job" && candidate.RefGUID == job.GUID {
			found = append(found, candidate)
		}
	}

	return found
}

func (service *ExecutionService) ByCall(call *core.Call) []*core.Execution {
	found := make([]*core.Execution, 0)

	for _, candidate := range service.storage {
		if candidate.RefType == "call" && candidate.RefGUID == call.GUID {
			found = append(found, candidate)
		}
	}

	return found
}

func (service *ExecutionService) BySchedule(schedule *core.Schedule) []*core.Execution {
	found := make([]*core.Execution, 0)

	for _, candidate := range service.storage {
		if candidate.ScheduleGUID == schedule.GUID {
			found = append(found, candidate)
		}
	}

	return found
}
