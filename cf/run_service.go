package cf

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
)

// taskPollInterval is how long waitForTask sleeps between asking Cloud
// Controller for a running task's state.
var taskPollInterval = 5 * time.Second

// taskGUIDAttempts and taskGUIDBackoff bound how often Execute tries to
// store a new task's GUID on its execution. The wait starts at
// taskGUIDBackoff and doubles after each failure.
var (
	taskGUIDAttempts = 5
	taskGUIDBackoff  = time.Second
)

type RunService struct {
	client CFClient
}

func NewRunService(client CFClient) *RunService {
	return &RunService{client}
}

func (service *RunService) Execute(services *core.Services, execution *core.Execution, executable core.Executable) {
	services.Workers.Submit(func() {
		defer core.RecoverRun(services, execution, "cf-run-service")

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
			return
		}

		// do real stuff
		startmsg := fmt.Sprintf("Starting job %s (%s)", job.Name, job.GUID)
		services.Logger.Info(tag, startmsg)

		request := resource.NewTaskCreateWithCommand(job.Command)
		request.WithName(job.Name).WithMemoryInMB(job.MemoryInMb).WithDiskInMB(job.DiskInMb).WithLogRateLimitInBytesPerSecond(job.LogRateInBps)

		task, err := service.client.CreateTask(context.Background(), job.AppGUID, request)
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
			service.storeTaskGUID(services, execution, task.GUID, tag)

			task, err = service.waitForTask(services, task)
			if err != nil {
				// There is no task to finalize, so the failure is the outcome.
				service.handleTaskFailure(services, execution, job, err)
			} else {
				services.Logger.Info(tag, fmt.Sprintf("task state before finalization: %s", task.State))

				service.finalizeTask(services, execution, job, task)
			}
		}

		finishmsg := fmt.Sprintf("Finishing job %s (%s)", job.Name, job.GUID)
		services.Logger.Info(tag, finishmsg)
	})
}

// storeTaskGUID records the task GUID on the execution, retrying with
// backoff. The task is already running in Cloud Controller, so when every
// attempt fails it logs the loss and lets the run go on to follow the task.
// It never replaces execution, because a failed update returns none.
func (service *RunService) storeTaskGUID(services *core.Services, execution *core.Execution, taskGUID string, tag string) {
	backoff := taskGUIDBackoff

	var err error
	for attempt := 1; attempt <= taskGUIDAttempts; attempt++ {
		if _, err = services.Executions.UpdateTaskGUID(execution, taskGUID); err == nil {
			return
		}

		if attempt == taskGUIDAttempts {
			break
		}

		services.Logger.Warn(tag, fmt.Sprintf(
			"cannot store task %s on execution %s (attempt %d of %d), retrying in %s: %v",
			taskGUID, execution.GUID, attempt, taskGUIDAttempts, backoff, err,
		))
		time.Sleep(backoff)
		backoff *= 2
	}

	services.Logger.Error(tag, fmt.Sprintf(
		"gave up storing task %s on execution %s after %d attempts, and the task is still followed to its end: %v",
		taskGUID, execution.GUID, taskGUIDAttempts, err,
	))
}

func (service *RunService) qq(services *core.Services, execution *core.Execution, tag string, message string) {
	services.Logger.Error(tag, message)
	services.Executions.UpdateMessage(execution, message)
	services.Executions.Fail(execution)
}

func (service *RunService) waitForTask(services *core.Services, task *resource.Task) (*resource.Task, error) {
	count := 0

	for task.State == "RUNNING" {
		// oh hey, the task is running, so let's wait a bit and see how it's
		// going

		services.Logger.Info("wait-for-task", fmt.Sprintf("waiting for task %s, iteration %d", task.GUID, count))
		time.Sleep(taskPollInterval)

		updatedTask, err := service.client.GetTask(context.Background(), task.GUID)
		if err != nil {
			// Even though the task was created, the API now says that we don't
			// get to know about it, so let's cry about it
			return nil, fmt.Errorf("api failure: %w", err)
		}

		services.Logger.Info("wait-for-task", fmt.Sprintf("got a task with state %s on iteration %d", updatedTask.State, count))

		task = updatedTask
		count = count + 1
	}

	return task, nil
}

func (service *RunService) handleTaskFailure(services *core.Services, execution *core.Execution, job *core.Job, err error) error {
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
	services.Logger.Error(tag, fmt.Sprintf("specific error: %s", err.Error()))

	return err
}

func (service *RunService) finalizeTask(services *core.Services, execution *core.Execution, job *core.Job, task *resource.Task) {
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

		failureReason := ""
		if task.Result.FailureReason != nil {
			failureReason = *task.Result.FailureReason
		}
		services.Executions.UpdateMessage(execution, failureReason)
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
