package http

import (
	"fmt"

	"github.com/ess/hype"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

type RunService struct {
}

func NewRunService() *RunService {
	return &RunService{}
}

func (service *RunService) Execute(services *core.Services, execution *core.Execution, executable core.Executable) {
	services.Workers.Submit(func() {
		defer core.RecoverRun(services, execution, "http-run-service")

		tag := "http-run-service"

		// A run that cannot mark its execution started stops here, rather
		// than send a request whose outcome the execution may never record.
		if _, err := services.Executions.Start(execution); err != nil {
			services.Logger.Error(tag, fmt.Sprintf("not running execution %s, because it could not be marked started: %v", execution.GUID, err))
			return
		}

		call, err := executable.ToCall()
		if err != nil {
			message := fmt.Sprintf(
				"cannot handle executables of type %s",
				executable.Type(),
			)

			services.Logger.Error(tag, message)
			services.Executions.UpdateMessage(execution, message)
			services.Executions.Fail(execution)
			return
		}

		// do real stuff
		startmsg := fmt.Sprintf("Starting call %s (%s)", call.Name, call.GUID)
		services.Logger.Info(tag, startmsg)

		driver, err := hype.New(call.URL)
		if err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("malformed URL for call %s (%s): %v", call.Name, call.GUID, err))
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
