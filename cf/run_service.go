package cf

import (
	"fmt"
	"time"

	cf "github.com/cloudfoundry-community/go-cfclient"
	"github.com/starkandwayne/scheduler-for-ocf/core"
)

type RunService struct {
	client Client
}

func NewRunService(client Client) *RunService {
	return &RunService{client}
}

func (service *RunService) Execute(services *core.Services, execution *core.Execution, executable core.Executable) {
	services.Workers.Submit(func() {
		services.Executions.Start(execution)
		tag := "cf-run-service"

		job, err := executable.ToJob()
		if err != nil {
			service.qq(
				services,
				execution,
				tag,
				fmt.Sprintf(
					"cannot handle executables of type %s",
					executable.Type(),
				),
			)
		}

		// do real stuff
		startmsg := fmt.Sprintf("Starting job %s (%s)", job.Name, job.GUID)
		services.Logger.Info(tag, startmsg)

		request := cf.TaskRequest{
			Command:          job.Command,
			MemoryInMegabyte: job.MemoryInMb,
			DiskInMegabyte:   job.DiskInMb,
		}

		task, err := service.client.CreateTask(request)
		if err != nil {
			// The cf api said "nope" when we tried to create the task, so let's
			// cry about it
			service.qq(
				services,
				execution,
				tag,
				fmt.Sprintf(
					"cannot create task for the job %s (%s)",
					job.Name,
					job.GUID,
				),
			)
		} else {
			// NOTE: this probably isn't necessary, but I need to double-check the
			// postgres.ExecutionService implementation to be sure
			execution, _ = services.Executions.UpdateTaskGUID(execution, task.GUID)

			for task.State == "RUNNING" {
				// oh hey, the task is running, so let's wait a bit and see how it's
				// going

				time.Sleep(5 * time.Second)

				task, err = service.client.GetTaskByGuid(task.GUID)
				if err != nil {
					// Even though the task was created, the API now says that we don't
					// get to know about it, so let's cry about it
					service.qq(
						services,
						execution,
						tag,
						fmt.Sprintf(
							"cannot get task for the job %s (%s)",
							job.Name,
							job.GUID,
						),
					)
				} else {
					if task.State == "FAILED" {
						// Uh-oh, the task ran, but it failed ... let's cry about it, but
						// not the same way that we cry about everything else
						message := fmt.Sprintf(
							"task failed for the job %s (%s)",
							job.Name,
							job.GUID,
						)

						services.Logger.Error(tag, message)
						services.Executions.UpdateMessage(execution, task.Result.FailureReason)
						services.Executions.Fail(execution)
					} else {
						// WE HAVE ACHIEVED A SUCCESSFUL TASK
						message := fmt.Sprintf(
							"task successfully completed for the job %s (%s)",
							job.Name,
							job.GUID,
						)

						services.Logger.Info(tag, message)
						services.Executions.Success(execution)
						break
					}
				}
			}
		}

		finishmsg := fmt.Sprintf("Finishing job %s (%s)", job.Name, job.GUID)
		services.Logger.Info(tag, finishmsg)
	})
}

func (service *RunService) qq(services *core.Services, execution *core.Execution, tag string, message string) {
	services.Logger.Error(tag, message)
	services.Executions.UpdateMessage(execution, message)
	services.Executions.Fail(execution)
}
