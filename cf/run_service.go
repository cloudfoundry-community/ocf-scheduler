package cf

import (
	"fmt"
	"time"

	cf "github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

type RunService struct {
	client *cf.Client
}

func NewRunService(client *cf.Client) *RunService {
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
			DropletGUID:      job.AppGUID,
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
			services.Logger.Error(
				"cf-run-service",
				fmt.Sprintf(
					"specific error: %s",
					err.Error(),
				),
			)
		} else {
			execution, err = services.Executions.UpdateTaskGUID(execution, task.GUID)
			for err != nil {
				// We literally can't continue until the execution has a task GUID
				execution, err = services.Executions.UpdateTaskGUID(execution, task.GUID)
			}

			task, err = service.waitForTask(services, task)
			if err != nil {
				service.handleTaskFailure(services, execution, job, task, err)
			}

			services.Logger.Info(tag, fmt.Sprintf("task state before finalization: %s", task.State))

			service.finalizeTask(services, execution, job, task)
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

func (service *RunService) waitForTask(services *core.Services, task cf.Task) (cf.Task, error) {
	count := 0

	for task.State == "RUNNING" {
		// oh hey, the task is running, so let's wait a bit and see how it's
		// going

		services.Logger.Info("wait-for-task", fmt.Sprintf("waiting for task %s, iteration %d", task.GUID, count))
		time.Sleep(5 * time.Second)

		updatedTask, err := service.client.GetTaskByGuid(task.GUID)
		if err != nil {
			// Even though the task was created, the API now says that we don't
			// get to know about it, so let's cry about it
			return updatedTask, fmt.Errorf("api failure")
		}

		services.Logger.Info("wait-for-task", fmt.Sprintf("got a task with state %s on iteration %d", updatedTask.State, count))

		task = updatedTask
		count = count + 1
	}

	return task, nil
}

func (service *RunService) handleTaskFailure(services *core.Services, execution *core.Execution, job *core.Job, task cf.Task, err error) error {
	tag := "cf-run-service"

	if err == nil {
		return nil
	}

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

	return err

}

func (service *RunService) finalizeTask(services *core.Services, execution *core.Execution, job *core.Job, task cf.Task) {
	tag := "cf-run-service"

	if task.State == "FAILED" {
		services.Logger.Error(
			tag,
			fmt.Sprintf(
				"task failed for job %s (%s)",
				job.Name,
				job.GUID,
			),
		)

		services.Executions.UpdateMessage(execution, task.Result.FailureReason)
		services.Executions.Fail(execution)

		return
	}

	// WE HAVE ACHIEVED A SUCCESSFUL TASK
	message := fmt.Sprintf(
		"task successfully completed for the job %s (%s)",
		job.Name,
		job.GUID,
	)

	services.Logger.Info(tag, message)
	services.Executions.Success(execution)
}
