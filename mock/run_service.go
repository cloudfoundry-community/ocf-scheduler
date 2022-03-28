package mock

import (
	"fmt"
	"time"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

type RunService struct {
}

func NewRunService() *RunService {
	return &RunService{}
}

func (service *RunService) Execute(services *core.Services, execution *core.Execution, executable core.Executable) {
	services.Workers.Submit(func() {
		tag := "mock-run-service"
		job, _ := executable.ToJob()

		services.Executions.Start(execution)

		services.Logger.Info(tag, fmt.Sprintf("Starting job %s (%s)", job.Name, job.GUID))
		time.Sleep(time.Second)

		services.Jobs.Success(job)

		services.Executions.UpdateMessage(execution, "ran by the mock runnner")
		services.Executions.Success(execution)

		services.Logger.Info(tag, fmt.Sprintf("Finishing job %s (%s)", job.Name, job.GUID))
	})
}
