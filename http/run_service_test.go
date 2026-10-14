package http

import (
	"strings"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/logger"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

// inlineWorkers runs each submitted function before Submit returns.
type inlineWorkers struct{}

func (inlineWorkers) Submit(work func()) { work() }
func (inlineWorkers) StopWait()          {}

// panickingExecutable panics when the runner asks for its call, standing in
// for any bug that panics inside a worker.
type panickingExecutable struct{}

func (panickingExecutable) Type() string                { return "call" }
func (panickingExecutable) ToJob() (*core.Job, error)   { return nil, nil }
func (panickingExecutable) ToCall() (*core.Call, error) { panic("call lookup blew up") }

// The worker pool does not recover panics, so a panic in a run would end the
// scheduler. The run recovers it and fails the execution instead.
func TestRunService_ExecuteRecoversAPanic(t *testing.T) {
	services := &core.Services{
		Workers:    inlineWorkers{},
		Executions: mock.NewExecutionService(),
		Logger:     logger.New(),
	}

	execution, err := services.Executions.Persist(&core.Execution{})
	if err != nil {
		t.Fatal(err)
	}

	NewRunService().Execute(services, execution, panickingExecutable{})

	if execution.State != "FAILED" {
		t.Errorf("execution state is %q, want FAILED", execution.State)
	}
	if !strings.Contains(execution.Message, "call lookup blew up") {
		t.Errorf("execution message is %q, want it to name the panic", execution.Message)
	}
}
