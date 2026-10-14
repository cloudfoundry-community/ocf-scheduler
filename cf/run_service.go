package cf

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
)

// taskPollInterval is how long waitForTask sleeps between asking Cloud
// Controller for a running task's state.
var taskPollInterval = 5 * time.Second

// taskPendingLimit is how long waitForTask follows a task that Cloud
// Controller still reports as PENDING. A task that has started and is
// RUNNING has no limit.
var taskPendingLimit = 15 * time.Minute

// errTaskNeverStarted is returned by waitForTask for a task that was still
// PENDING when taskPendingLimit ran out.
var errTaskNeverStarted = errors.New("the task never started")

// taskGUIDAttempts and taskGUIDBackoff bound how often Execute tries to
// mark an execution started and to store a new task's GUID on it. The wait
// starts at taskGUIDBackoff and doubles after each failure.
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

		tag := "cf-run-service"

		// A run that cannot mark its execution started is skipped, rather
		// than send a task whose outcome the execution may never record.
		if !core.StartRun(services, execution, tag, taskGUIDAttempts, taskGUIDBackoff) {
			return
		}

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
			switch {
			case errors.Is(err, errTaskNeverStarted):
				message, finished := service.cancelUnstartedTask(services, job, task, tag)
				if finished != nil {
					service.finalizeTask(services, execution, job, finished)
				} else {
					service.qq(services, execution, tag, message)
				}
			case err != nil:
				// There is no task to finalize, so the failure is the outcome.
				service.handleTaskFailure(services, execution, job, err)
			default:
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

// taskUnfinished reports whether a task in state may still change, so
// waitForTask keeps polling it.
func taskUnfinished(state string) bool {
	switch state {
	case "PENDING", "RUNNING", "CANCELING":
		return true
	}

	return false
}

// waitForTask polls the task until it leaves the pending, running, and
// canceling states, and returns it in the state it ended in. A task still
// PENDING once taskPendingLimit has passed is returned as it was last seen,
// with errTaskNeverStarted.
func (service *RunService) waitForTask(services *core.Services, task *resource.Task) (*resource.Task, error) {
	count := 0
	started := time.Now()

	for taskUnfinished(task.State) {
		if task.State == "PENDING" && time.Since(started) >= taskPendingLimit {
			return task, errTaskNeverStarted
		}

		// oh hey, the task is still going, so let's wait a bit and see how
		// it's doing

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

// cancelUnstartedTask cancels a task that stayed PENDING past
// taskPendingLimit, so that Cloud Controller cannot start it later with no
// execution left to record its outcome. It returns the message for the
// failed execution, which says whether the cancel went through.
//
// Cloud Controller refuses to cancel a task that ended since the last poll,
// so after a failed cancel the task is read once more. When it turns out to
// be SUCCEEDED or FAILED, it is returned as the outcome to record in place of
// the message.
func (service *RunService) cancelUnstartedTask(services *core.Services, job *core.Job, task *resource.Task, tag string) (string, *resource.Task) {
	message := fmt.Sprintf(
		"task %s for the job %s (%s) never started, because it was still PENDING after %s",
		task.GUID,
		job.Name,
		job.GUID,
		taskPendingLimit,
	)

	if _, err := service.client.CancelTask(context.Background(), task.GUID); err != nil {
		services.Logger.Error(tag, fmt.Sprintf("cannot cancel task %s for the job %s (%s): %v", task.GUID, job.Name, job.GUID, err))

		latest, getErr := service.client.GetTask(context.Background(), task.GUID)
		if getErr == nil && latest != nil && (latest.State == "SUCCEEDED" || latest.State == "FAILED") {
			return "", latest
		}

		return fmt.Sprintf("%s, and the scheduler could not cancel it, so Cloud Controller may still run it: %v", message, err), nil
	}

	return message + ", so the scheduler cancelled it", nil
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

// finalizeTask records how a finished task ended. Only SUCCEEDED counts as
// success, and any state Cloud Controller adds later counts as a failure.
func (service *RunService) finalizeTask(services *core.Services, execution *core.Execution, job *core.Job, task *resource.Task) {
	tag := "cf-run-service"

	switch task.State {
	case "SUCCEEDED":
		// WE HAVE ACHIEVED A SUCCESSFUL TASK
		message := fmt.Sprintf(
			"task successfully completed for the job %s (%s)",
			job.Name,
			job.GUID,
		)

		services.Logger.Info(tag, message)
		services.Executions.Success(execution)
	case "FAILED":
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
	default:
		service.qq(
			services,
			execution,
			tag,
			fmt.Sprintf(
				"task for the job %s (%s) ended in the unknown state %s",
				job.Name,
				job.GUID,
				task.State,
			),
		)
	}
}
