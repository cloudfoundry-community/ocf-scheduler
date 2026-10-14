package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

// stallingExecutions fails the first startFailures calls to Start.
type stallingExecutions struct {
	*mock.ExecutionService
	startFailures, starts int
}

func (executions *stallingExecutions) Start(execution *core.Execution) (*core.Execution, error) {
	executions.starts++
	if executions.starts <= executions.startFailures {
		return nil, errors.New("database unavailable")
	}
	return executions.ExecutionService.Start(execution)
}

// A brief database error while marking the execution started costs only a
// retry. A run that never gets marked started sends no request, because the
// execution might never record how the call went, and its execution is
// failed rather than left PENDING.
func TestRunService_ExecuteRetriesStartingTheExecution(t *testing.T) {
	savedAttempts, savedBackoff := startAttempts, startBackoff
	startAttempts, startBackoff = 3, time.Millisecond
	defer func() { startAttempts, startBackoff = savedAttempts, savedBackoff }()

	tests := []struct {
		name          string
		startFailures int
		wantRequests  int32
		wantState     string
	}{
		{name: "a passing failure", startFailures: 1, wantRequests: 1, wantState: "SUCCEEDED"},
		{name: "a failure that outlasts every attempt", startFailures: 100, wantRequests: 0, wantState: "FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
			}))
			defer server.Close()

			executions := &stallingExecutions{ExecutionService: mock.NewExecutionService(), startFailures: tt.startFailures}
			services := &core.Services{Workers: inlineWorkers{}, Executions: executions, Logger: logger.New()}
			execution, err := executions.Persist(&core.Execution{})
			if err != nil {
				t.Fatal(err)
			}

			NewRunService().Execute(services, execution, &core.Call{GUID: "call-guid", Name: "ping", URL: server.URL})

			if n := requests.Load(); n != tt.wantRequests {
				t.Errorf("sent %d requests, want %d", n, tt.wantRequests)
			}
			if execution.State != tt.wantState {
				t.Errorf("execution state is %q, want %q", execution.State, tt.wantState)
			}
		})
	}
}
