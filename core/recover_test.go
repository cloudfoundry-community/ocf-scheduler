package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

// stuckExecutions cannot record anything on an execution.
type stuckExecutions struct {
	*mock.ExecutionService
}

func (stuckExecutions) UpdateMessage(*core.Execution, string) (*core.Execution, error) {
	return nil, errors.New("database unavailable")
}

func (stuckExecutions) Fail(*core.Execution) (*core.Execution, error) {
	return nil, errors.New("database unavailable")
}

// When a run panics and the execution cannot be failed, RecoverRun says so,
// because otherwise nothing shows that the execution was left unfinished.
func TestRecoverRunLogsAFailureItCannotRecord(t *testing.T) {
	log := &recordingLog{}
	services := &core.Services{Executions: stuckExecutions{mock.NewExecutionService()}, Logger: log}
	execution := &core.Execution{GUID: "execution-guid"}

	func() {
		defer core.RecoverRun(services, execution, "test-run")
		panic("run blew up")
	}()

	var message, fail bool
	for _, line := range log.lines {
		message = message || strings.Contains(line, "could not record the panic on execution execution-guid")
		fail = fail || strings.Contains(line, "could not fail execution execution-guid")
	}
	if !message || !fail {
		t.Errorf("logged %q, want lines saying the message and the failure were not recorded", log.lines)
	}
}

// A panic in a run with no execution is still recovered and logged, rather
// than turned into a second panic that ends the scheduler.
func TestRecoverRunHandlesANilExecution(t *testing.T) {
	log := &recordingLog{}
	services := &core.Services{Executions: mock.NewExecutionService(), Logger: log}

	escaped := func() (escaped any) {
		defer func() { escaped = recover() }()
		func() {
			defer core.RecoverRun(services, nil, "test-run")
			panic("run blew up")
		}()
		return nil
	}()

	if escaped != nil {
		t.Fatalf("a panic escaped RecoverRun: %v", escaped)
	}
	if len(log.lines) != 1 || !strings.Contains(log.lines[0], "run blew up") {
		t.Errorf("logged %q, want one line naming the panic", log.lines)
	}
}
