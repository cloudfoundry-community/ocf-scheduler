package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_LoadExecutionCollection(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	executionService := mock.NewExecutionService()
	logService := mock.NewLogService()
	services := &core.Services{Executions: executionService, Logger: logService}
	guid, _ := core.GenGUID()
	executable := &core.Job{GUID: guid, Name: "Bob"}

	t.Run("when the provided executable is nil", func(t *testing.T) {
		input := core.
			NewInput(services)

		t.Run("the op fails due to a type mismatch", func(t *testing.T) {
			result := LoadExecutionCollection(input)

			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.ExecutableTypeMismatch {
				t.Errorf("expected an executable type mismatch, got %s", cause)
			}
		})
	})

	t.Run("when the executable has executionss", func(t *testing.T) {
		executionService.Reset()
		input := core.
			NewInput(services).
			WithExecutable(executable)

		for i := 0; i < 20; i++ {
			seed := &core.Execution{RefGUID: guid, RefType: "job"}

			if _, err := executionService.Persist(dummyExecution(seed)); err != nil {
				t.Errorf("could not persist execution collection")
			}
		}

		result := LoadExecutionCollection(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting execution collection contains the proper executions", func(t *testing.T) {
			payload := core.Inputify(result.Value())
			count := len(payload.ExecutionCollection)

			if count != 20 {
				t.Errorf("expected 20 executions, got %d", count)
			}
		})
	})

	t.Run("when the executable has no executions", func(t *testing.T) {
		executionService.Reset()
		input := core.
			NewInput(services).
			WithExecutable(executable)

		result := LoadExecutionCollection(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting execution collection is empty", func(t *testing.T) {
			payload := core.Inputify(result.Value())
			count := len(payload.ExecutionCollection)

			if count != 0 {
				t.Errorf("expected no executions, got %d", count)
			}
		})
	})
}
