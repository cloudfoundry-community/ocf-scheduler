package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/cron"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_ValidateScheduleExpression(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	logScope := "ops.validate-schedule-expression"
	logService := mock.NewLogService()
	cronService := cron.NewCronService(logService)
	services := &core.Services{Cron: cronService, Logger: logService}

	t.Run("when given a valid cron expression", func(t *testing.T) {
		schedule := dummySchedule(nil)
		input := core.NewInput(services).
			WithSchedule(schedule)

		result := ValidateScheduleExpression(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})
	})

	t.Run("when given an invalid cron expression", func(t *testing.T) {
		schedule := dummySchedule(nil)
		schedule.Expression = "a b c d e f g h i"
		input := core.NewInput(services).
			WithSchedule(schedule)

		result := ValidateScheduleExpression(input)

		t.Run("the op is a failure", func(t *testing.T) {
			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.ScheduleExpressionInvalid {
				t.Errorf("expected error %s, got %s", failures.ScheduleExpressionInvalid, cause)
			}
		})

		t.Run("the invalid cron expression is logged", func(t *testing.T) {
			if !logService.ReceivedError(logScope, "schedule cron expression invalid") {
				t.Errorf("expected the bad cron expression to be logged")
			}
		})
	})

	t.Run("when given no schedule", func(t *testing.T) {
		input := core.NewInput(services)

		result := ValidateScheduleExpression(input)

		t.Run("the op is a failure", func(t *testing.T) {
			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.ScheduleNotProvided {
				t.Errorf("expected %s, got %s", failures.ScheduleNotProvided, cause)
			}
		})
	})
}
