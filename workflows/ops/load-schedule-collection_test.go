package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_LoadScheduleCollection(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	scheduleService := mock.NewScheduleService()
	logService := mock.NewLogService()
	services := &core.Services{Schedules: scheduleService, Logger: logService}
	guid, _ := core.GenGUID()
	executable := &core.Job{GUID: guid, Name: "Bob"}

	t.Run("when the provided executable is nil", func(t *testing.T) {
		input := core.
			NewInput(services)

		t.Run("the op fails due to a type mismatch", func(t *testing.T) {
			result := LoadScheduleCollection(input)

			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.ExecutableTypeMismatch {
				t.Errorf("expected an executable type mismatch, got %s", cause)
			}
		})
	})

	t.Run("when the executable has schedules", func(t *testing.T) {
		scheduleService.Reset()
		input := core.
			NewInput(services).
			WithExecutable(executable)

		for i := 0; i < 20; i++ {
			seed := &core.Schedule{RefGUID: guid, RefType: "job"}

			if _, err := scheduleService.Persist(dummySchedule(seed)); err != nil {
				t.Errorf("could not persist schedule collection")
			}
		}

		result := LoadScheduleCollection(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting schedule collection contains the proper schedules", func(t *testing.T) {
			payload := core.Inputify(result.Value())
			count := len(payload.ScheduleCollection)

			if count != 20 {
				t.Errorf("expected 20 schedules, got %d", count)
			}
		})
	})

	t.Run("when the executable has no schedules", func(t *testing.T) {
		scheduleService.Reset()
		input := core.
			NewInput(services).
			WithExecutable(executable)

		result := LoadScheduleCollection(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting schedule collection is empty", func(t *testing.T) {
			payload := core.Inputify(result.Value())
			count := len(payload.ScheduleCollection)

			if count != 0 {
				t.Errorf("expected no schedules, got %d", count)
			}
		})
	})
}
