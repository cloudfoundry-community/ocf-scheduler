package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

// recordingLog keeps every line logged at warn or above.
type recordingLog struct {
	lines []string
}

func (log *recordingLog) Trace(string, string) {}
func (log *recordingLog) Debug(string, string) {}
func (log *recordingLog) Info(string, string)  {}
func (log *recordingLog) Warn(tag, message string) {
	log.lines = append(log.lines, tag+": "+message)
}
func (log *recordingLog) Error(tag, message string) {
	log.lines = append(log.lines, tag+": "+message)
}
func (log *recordingLog) Fatal(tag, message string) {
	log.lines = append(log.lines, tag+": "+message)
}
func (log *recordingLog) Panic(tag, message string) {
	log.lines = append(log.lines, tag+": "+message)
}

// lostExecutions cannot store an execution, as when the database is down.
type lostExecutions struct {
	*mock.ExecutionService
}

func (lostExecutions) Persist(*core.Execution) (*core.Execution, error) {
	return nil, errors.New("database unavailable")
}

// countingRunner counts the runs it is handed.
type countingRunner struct {
	executions []*core.Execution
}

func (runner *countingRunner) Execute(_ *core.Services, execution *core.Execution, _ core.Executable) {
	runner.executions = append(runner.executions, execution)
}

// A scheduled run whose execution cannot be stored is skipped with one log
// line, rather than handed to the runner with no execution to record on.
func TestScheduledRunsSkipARunWhoseExecutionCannotBeStored(t *testing.T) {
	schedule := &core.Schedule{GUID: "schedule-guid"}
	for _, tt := range []struct {
		name string
		run  func(services *core.Services) core.Runnable
	}{
		{name: "job", run: func(services *core.Services) core.Runnable {
			return core.NewJobRun(&core.Job{GUID: "job-guid"}, schedule, services)
		}},
		{name: "call", run: func(services *core.Services) core.Runnable {
			return core.NewCallRun(&core.Call{GUID: "call-guid"}, schedule, services)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			log := &recordingLog{}
			runner := &countingRunner{}
			services := &core.Services{Executions: lostExecutions{mock.NewExecutionService()}, Runner: runner, Logger: log}

			tt.run(services).Run()

			if len(runner.executions) != 0 {
				t.Errorf("the runner was handed %d runs, want none", len(runner.executions))
			}
			if len(log.lines) != 1 || !strings.Contains(log.lines[0], "schedule-guid") {
				t.Errorf("logged %q, want one line naming the schedule", log.lines)
			}

			// With the database back, the next run goes ahead.
			services.Executions = mock.NewExecutionService()
			tt.run(services).Run()

			if len(runner.executions) != 1 || runner.executions[0] == nil {
				t.Errorf("the runner was handed %v, want one run with its execution", runner.executions)
			}
		})
	}
}
