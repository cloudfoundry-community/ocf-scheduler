package mock

import (
	"time"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

type RunService struct {
}

func NewRunService() *RunService {
	return &RunService{}
}

func (service *RunService) ExecuteJob(services *core.Services, execution *core.Execution, job *core.Job) {
	services.Workers.Submit(func() {
		services.Executions.Start(execution)
		time.Sleep(time.Second)
		services.Executions.UpdateMessage(execution, "ran by the mock runnner")
		services.Executions.Success(execution)
	})
}

func (service *RunService) ExecuteCall(services *core.Services, execution *core.Execution, call *core.Call) {
	services.Workers.Submit(func() {
		services.Executions.Start(execution)
		time.Sleep(time.Second)
		services.Executions.UpdateMessage(execution, "ran by the mock runnner")
		services.Executions.Success(execution)
	})
}
