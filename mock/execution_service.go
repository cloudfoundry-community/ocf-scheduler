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
	execution.ExecutionStartTime = time.Now().UTC().String()

	return service.update(execution)
}

func (service *ExecutionService) finish(execution *core.Execution, state string) (*core.Execution, error) {
	execution.ExecutionEndTime = time.Now().UTC().String()
	execution.State = state

	return service.update(execution)
}

func (service *ExecutionService) Success(execution *core.Execution) (*core.Execution, error) {
	return service.finish(execution, "SUCCESS")
}

func (service *ExecutionService) Fail(execution *core.Execution) (*core.Execution, error) {
	return service.finish(execution, "FAILURE")
}

//func (service *ExecutionService) Get(guid string) (*core.Execution, error) {
//candidate, found := service.storage[guid]
//if !found {
//return nil, fmt.Errorf("not found")
//}

//return candidate, nil
//}
