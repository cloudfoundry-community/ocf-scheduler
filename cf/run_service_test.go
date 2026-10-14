package cf_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry/go-cfclient/v3/resource"

	"github.com/cloudfoundry-community/ocf-scheduler/cf"
	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/logger"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

// inlineWorkers runs each submitted function before Submit returns.
type inlineWorkers struct{}

func (inlineWorkers) Submit(work func()) { work() }
func (inlineWorkers) StopWait()          {}

// An executable that is not a job fails the execution and stops there,
// instead of going on to use a job that does not exist.
func TestRunService_ExecuteStopsWhenTheExecutableIsNotAJob(t *testing.T) {
	services := &core.Services{
		Workers:    inlineWorkers{},
		Executions: mock.NewExecutionService(),
		Logger:     logger.New(),
	}

	execution, err := services.Executions.Persist(&core.Execution{})
	if err != nil {
		t.Fatal(err)
	}

	// A nil client is never reached, because the run stops before it.
	cf.NewRunService(nil).Execute(services, execution, &core.Call{Name: "ping"})

	if execution.State != "FAILED" {
		t.Errorf("execution state is %q, want FAILED", execution.State)
	}
}

// lostTaskClient creates tasks that are still running and then cannot read
// them back, as when Cloud Controller fails after the task starts.
type lostTaskClient struct {
	*mock.CFClient
}

func (lostTaskClient) GetTask(_ context.Context, _ string) (*resource.Task, error) {
	return nil, errors.New("cloud controller unavailable")
}

func newRunServices(t *testing.T) (*core.Services, *core.Execution) {
	t.Helper()
	services := &core.Services{
		Workers:    inlineWorkers{},
		Executions: mock.NewExecutionService(),
		Logger:     logger.New(),
	}

	execution, err := services.Executions.Persist(&core.Execution{})
	if err != nil {
		t.Fatal(err)
	}

	return services, execution
}

// A task that Cloud Controller cannot report on fails the execution and
// stops the run there, instead of reading the state of a task it never got.
func TestRunService_ExecuteStopsWhenTheTaskCannotBeRead(t *testing.T) {
	defer cf.SetTaskPollInterval(time.Millisecond)()

	client, err := mock.NewCFClient()
	if err != nil {
		t.Fatal(err)
	}
	services, execution := newRunServices(t)
	job := &core.Job{GUID: "job-guid", Name: "nightly", Command: "true", AppGUID: "app-guid"}

	cf.NewRunService(lostTaskClient{client}).Execute(services, execution, job)

	if execution.State != "FAILED" {
		t.Errorf("execution state is %q, want FAILED", execution.State)
	}
	if want := "cannot get task for the job nightly (job-guid)"; execution.Message != want {
		t.Errorf("execution message is %q, want %q", execution.Message, want)
	}
}

// panickingClient panics when asked to create a task, standing in for any
// bug that panics inside a worker.
type panickingClient struct {
	*mock.CFClient
}

func (panickingClient) CreateTask(_ context.Context, _ string, _ *resource.TaskCreate) (*resource.Task, error) {
	panic("task creation blew up")
}

// The worker pool does not recover panics, so a panic in a run would end the
// scheduler. The run recovers it and fails the execution instead.
func TestRunService_ExecuteRecoversAPanic(t *testing.T) {
	client, err := mock.NewCFClient()
	if err != nil {
		t.Fatal(err)
	}
	services, execution := newRunServices(t)
	job := &core.Job{GUID: "job-guid", Name: "nightly", Command: "true", AppGUID: "app-guid"}

	cf.NewRunService(panickingClient{client}).Execute(services, execution, job)

	if execution.State != "FAILED" {
		t.Errorf("execution state is %q, want FAILED", execution.State)
	}
	if !strings.Contains(execution.Message, "task creation blew up") {
		t.Errorf("execution message is %q, want it to name the panic", execution.Message)
	}
}

// flakyExecutions fails the first failures calls to UpdateTaskGUID the way
// the postgres service does, with no execution and an error.
type flakyExecutions struct {
	*mock.ExecutionService
	failures int
	calls    int
}

func (executions *flakyExecutions) UpdateTaskGUID(execution *core.Execution, taskGUID string) (*core.Execution, error) {
	executions.calls++
	if executions.calls <= executions.failures {
		return nil, errors.New("database unavailable")
	}
	return executions.ExecutionService.UpdateTaskGUID(execution, taskGUID)
}

func TestRunService_ExecuteRetriesStoringTheTaskGUID(t *testing.T) {
	defer cf.SetTaskPollInterval(time.Millisecond)()

	tests := []struct {
		name      string
		failures  int
		wantCalls int
	}{
		{name: "a passing failure", failures: 2, wantCalls: 3},
		{name: "a failure that outlasts every attempt", failures: 100, wantCalls: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer cf.SetTaskGUIDRetries(4, time.Millisecond)()

			client, err := mock.NewCFClient()
			if err != nil {
				t.Fatal(err)
			}
			executions := &flakyExecutions{ExecutionService: mock.NewExecutionService(), failures: tt.failures}
			services := &core.Services{
				Workers:    inlineWorkers{},
				Executions: executions,
				Logger:     logger.New(),
			}
			execution, err := executions.Persist(&core.Execution{})
			if err != nil {
				t.Fatal(err)
			}
			job := &core.Job{GUID: "job-guid", Name: "nightly", Command: "true", AppGUID: "app-guid"}

			cf.NewRunService(client).Execute(services, execution, job)

			if executions.calls != tt.wantCalls {
				t.Errorf("UpdateTaskGUID was called %d times, want %d", executions.calls, tt.wantCalls)
			}
			// The task is already running in Cloud Controller, so the run
			// follows it to the end even when its GUID could not be stored.
			if execution.State != "SUCCEEDED" {
				t.Errorf("execution state is %q, want SUCCEEDED", execution.State)
			}
		})
	}
}

// scriptedTaskClient creates a task in the first of states and reports the
// rest, one per GetTask.
type scriptedTaskClient struct {
	*mock.CFClient
	states []string
	reads  int
}

func (client *scriptedTaskClient) CreateTask(_ context.Context, _ string, _ *resource.TaskCreate) (*resource.Task, error) {
	return &resource.Task{State: client.states[0], Resource: resource.Resource{GUID: "task-guid"}}, nil
}

func (client *scriptedTaskClient) GetTask(_ context.Context, guid string) (*resource.Task, error) {
	client.reads++
	if client.reads >= len(client.states) {
		return nil, errors.New("the task was read after it finished")
	}
	return &resource.Task{State: client.states[client.reads], Resource: resource.Resource{GUID: guid}}, nil
}

// The run waits while a task is pending, running, or canceling. Only a task
// that succeeded counts as success, and a state the scheduler does not know
// counts as a failure that names it.
func TestRunService_ExecuteFollowsATaskUntilItFinishes(t *testing.T) {
	defer cf.SetTaskPollInterval(time.Millisecond)()

	tests := []struct {
		name        string
		states      []string
		wantState   string
		wantMessage string
	}{
		{name: "pending then succeeded", states: []string{"PENDING", "RUNNING", "SUCCEEDED"}, wantState: "SUCCEEDED"},
		{name: "canceled", states: []string{"PENDING", "CANCELING", "FAILED"}, wantState: "FAILED"},
		{name: "an unknown state", states: []string{"RUNNING", "MISPLACED"}, wantState: "FAILED", wantMessage: "MISPLACED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := mock.NewCFClient()
			if err != nil {
				t.Fatal(err)
			}
			scripted := &scriptedTaskClient{CFClient: client, states: tt.states}
			services, execution := newRunServices(t)
			job := &core.Job{GUID: "job-guid", Name: "nightly", Command: "true", AppGUID: "app-guid"}

			cf.NewRunService(scripted).Execute(services, execution, job)

			if want := len(tt.states) - 1; scripted.reads != want {
				t.Errorf("the task was read %d times, want %d", scripted.reads, want)
			}
			if execution.State != tt.wantState {
				t.Errorf("execution state is %q, want %q", execution.State, tt.wantState)
			}
			if !strings.Contains(execution.Message, tt.wantMessage) {
				t.Errorf("execution message is %q, want it to name %q", execution.Message, tt.wantMessage)
			}
		})
	}
}

// countingClient counts the tasks it is asked to create.
type countingClient struct {
	*mock.CFClient
	created int
}

func (client *countingClient) CreateTask(ctx context.Context, appGUID string, request *resource.TaskCreate) (*resource.Task, error) {
	client.created++
	return client.CFClient.CreateTask(ctx, appGUID, request)
}

// unstartableExecutions cannot mark an execution started.
type unstartableExecutions struct {
	*mock.ExecutionService
}

func (unstartableExecutions) Start(*core.Execution) (*core.Execution, error) {
	return nil, errors.New("database unavailable")
}

// lostExecutions cannot store an execution at all.
type lostExecutions struct {
	*mock.ExecutionService
}

func (lostExecutions) Persist(*core.Execution) (*core.Execution, error) {
	return nil, errors.New("database unavailable")
}

// A run whose execution cannot be marked started creates no task, because
// the execution might never record what the task did.
func TestRunService_ExecuteCreatesNoTaskWhenTheExecutionCannotStart(t *testing.T) {
	mockClient, err := mock.NewCFClient()
	if err != nil {
		t.Fatal(err)
	}
	client := &countingClient{CFClient: mockClient}
	executions := unstartableExecutions{mock.NewExecutionService()}
	services := &core.Services{Workers: inlineWorkers{}, Executions: executions, Logger: logger.New()}
	execution, err := executions.Persist(&core.Execution{})
	if err != nil {
		t.Fatal(err)
	}
	job := &core.Job{GUID: "job-guid", Name: "nightly", Command: "true", AppGUID: "app-guid"}

	cf.NewRunService(client).Execute(services, execution, job)

	if client.created != 0 {
		t.Errorf("created %d tasks, want none", client.created)
	}
}

// A scheduled job run whose execution cannot be stored creates no task.
func TestJobRun_CreatesNoTaskWhenTheExecutionCannotBeStored(t *testing.T) {
	mockClient, err := mock.NewCFClient()
	if err != nil {
		t.Fatal(err)
	}
	client := &countingClient{CFClient: mockClient}
	services := &core.Services{
		Workers:    inlineWorkers{},
		Executions: lostExecutions{mock.NewExecutionService()},
		Runner:     cf.NewRunService(client),
		Logger:     logger.New(),
	}
	job := &core.Job{GUID: "job-guid", Name: "nightly", Command: "true", AppGUID: "app-guid"}

	core.NewJobRun(job, &core.Schedule{GUID: "schedule-guid"}, services).Run()

	if client.created != 0 {
		t.Errorf("created %d tasks, want none", client.created)
	}
}
