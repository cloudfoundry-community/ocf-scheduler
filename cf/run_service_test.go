package cf_test

import (
	"context"
	"errors"
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
