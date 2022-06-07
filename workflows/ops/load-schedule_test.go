package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_LoadSchedule(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	logScope := "ops.load-schedule"
	scheduleService := mock.NewScheduleService()
	logService := mock.NewLogService()
	services := &core.Services{Schedules: scheduleService, Logger: logService}
	guid, _ := core.GenGUID()
	schedule := dummySchedule(&core.Schedule{GUID: guid})

	t.Run("when a guid is provided", func(t *testing.T) {
		input := core.
			NewInput(services).
			WithScheduleGUID(guid)

		t.Run("and the schedule exists", func(t *testing.T) {
			schedule, err := scheduleService.Persist(schedule)
			if err != nil {
				t.Errorf("could not persist schedule: %s", err.Error())
			}

			input = input.WithScheduleGUID(schedule.GUID)

			result := LoadSchedule(input)

			t.Run("the op is a success", func(t *testing.T) {
				if result.Failure() {
					cause := core.Causify(result.Error())
					t.Errorf("expected a success, got failure: %s, %s", cause, logService.ErrorsFor(logScope))
				}
			})
		})

		t.Run("but the schedule doesn't exist", func(t *testing.T) {
			logService.Reset()
			scheduleService.Delete(schedule)

			result := LoadSchedule(input)

			t.Run("the op logs the issue with finding the schedule", func(t *testing.T) {
				errorLogged := logService.ReceivedError(
					logScope,
					"could not find schedule with guid "+schedule.GUID,
				)

				if !errorLogged {
					t.Errorf(
						"expected a logged error regarding the unknown schedule %s",
						schedule.GUID,
					)
				}
			})

			t.Run("the op is a failure", func(t *testing.T) {
				if result.Success() {
					t.Errorf("expected a failure, got success")
				}

				cause := core.Causify(result.Error())
				if cause != failures.NoSuchSchedule {
					t.Errorf("expected '%s', got '%s'", failures.NoSuchSchedule, cause)
				}
			})
		})
	})

	t.Run("when no guid is provided", func(t *testing.T) {
		logService.Reset()
		input := core.
			NewInput(services)

		result := LoadSchedule(input)

		t.Run("the op logs an error", func(t *testing.T) {
			if !logService.ReceivedError(logScope, "no schedule guid provided") {
				t.Errorf("expected a logged error regarding the missing guid")
			}
		})

		t.Run("the op is a failure", func(t *testing.T) {
			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.NoScheduleGUID {
				t.Errorf("expected '%s', got '%s'", failures.NoScheduleGUID, cause)
			}
		})
	})

}
