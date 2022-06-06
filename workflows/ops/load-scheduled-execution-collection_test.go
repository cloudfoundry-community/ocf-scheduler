package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_LoadScheduledExecutionCollection(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	executionService := mock.NewExecutionService()
	logService := mock.NewLogService()
	services := &core.Services{Executions: executionService, Logger: logService}
	schedule := dummySchedule(nil)
	jobGUID, _ := core.GenGUID()

	t.Run("when the provided schedule is nil", func(t *testing.T) {
		input := core.
			NewInput(services)

		t.Run("the op fails due to the missing schedule", func(t *testing.T) {
			result := LoadScheduledExecutionCollection(input)

			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.ScheduleNotProvided {
				t.Errorf("expected a schedule not provided, got %s", cause)
			}
		})
	})

	t.Run("when the execution has executions", func(t *testing.T) {
		executionService.Reset()
		input := core.
			NewInput(services).
			WithSchedule(schedule)

		for i := 0; i < 20; i++ {
			seed := &core.Execution{
				RefGUID:      jobGUID,
				RefType:      "job",
				ScheduleGUID: schedule.GUID,
			}

			if _, err := executionService.Persist(dummyExecution(seed)); err != nil {
				t.Errorf("could not persist execution collection")
			}
		}

		result := LoadScheduledExecutionCollection(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting execution collection contains the proper schedules", func(t *testing.T) {
			payload := core.Inputify(result.Value())
			count := len(payload.ExecutionCollection)

			if count != 20 {
				t.Errorf("expected 20 executions, got %d", count)
			}
		})
	})

	t.Run("when the schedule has no executions", func(t *testing.T) {
		executionService.Reset()
		input := core.
			NewInput(services).
			WithSchedule(schedule)

		result := LoadScheduledExecutionCollection(input)

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
