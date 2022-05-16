package http

import (
	"fmt"

	"github.com/ess/hype"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

type RunService struct {
}

func NewRunService() *RunService {
	return &RunService{}
}

func (service *RunService) Execute(services *core.Services, execution *core.Execution, executable core.Executable) {
	services.Workers.Submit(func() {
		services.Executions.Start(execution)
		tag := "http-run-service"

		call, err := executable.ToCall()
		if err != nil {
			message := fmt.Sprintf(
				"cannot handle executables of type %s",
				executable.Type(),
			)

			services.Logger.Error(tag, message)
			services.Executions.UpdateMessage(execution, message)
			services.Executions.Fail(execution)
		}

		// do real stuff
		startmsg := fmt.Sprintf("Starting call %s (%s)", call.Name, call.GUID)
		services.Logger.Info(tag, startmsg)

		driver, err := hype.New(call.URL)
		if err != nil {
			services.Executions.UpdateMessage(execution, "failed due to malformed URL")
			services.Executions.Fail(execution)
		} else {

			response := driver.
				Post("", nil, nil).
				WithHeaderSet(hype.NewHeader("Authorization", call.AuthHeader)).
				Response()

			if response.Okay() {
				services.Executions.UpdateMessage(execution, "POST success")
				services.Executions.Success(execution)
			} else {
				services.Executions.UpdateMessage(execution, response.Error().Error())
				services.Executions.Fail(execution)
			}
		}

		finishmsg := fmt.Sprintf("Finishing call %s (%s)", call.Name, call.GUID)
		services.Logger.Info(tag, finishmsg)
	})
}
