package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_LoadCall(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	logScope := "ops.load-call"
	callService := mock.NewCallService()
	logService := mock.NewLogService()
	services := &core.Services{Calls: callService, Logger: logService}
	guid, _ := core.GenGUID()
	call := dummyCall(&core.Call{GUID: guid, Name: "load-call-test"})

	t.Run("when a guid is provided", func(t *testing.T) {
		input := core.
			NewInput(services).
			WithGUID(guid)

		t.Run("and the call exists", func(t *testing.T) {
			call, err := callService.Persist(call)
			if err != nil {
				t.Errorf("could not persist call: %s", err.Error())
			}

			input = input.WithGUID(call.GUID)

			result := LoadCall(input)

			t.Run("the op is a success", func(t *testing.T) {
				if result.Failure() {
					cause := core.Causify(result.Error())
					t.Errorf("expected a success, got failure: %s, %s", cause, logService.ErrorsFor(logScope))
				}
			})

			t.Run("the payload has an executable call", func(t *testing.T) {
				payload := core.Inputify(result.Value())
				if payload.Executable == nil {
					t.Errorf("expected an executable, got nil")
				}

				_, err := payload.Executable.ToCall()
				if err != nil {
					t.Errorf("expected executable to be a call")
				}
			})
		})

		t.Run("but the call doesn't exist", func(t *testing.T) {
			logService.Reset()
			callService.Delete(call)

			result := LoadCall(input)

			t.Run("the op logs the issue with finding the call", func(t *testing.T) {
				errorLogged := logService.ReceivedError(
					logScope,
					"could not find call with guid "+call.GUID,
				)

				if !errorLogged {
					t.Errorf(
						"expected a logged error regarding the unknown call %s",
						call.GUID,
					)
				}
			})

			t.Run("the op is a failure", func(t *testing.T) {
				if result.Success() {
					t.Errorf("expected a failure, got success")
				}

				cause := core.Causify(result.Error())
				if cause != failures.NoSuchCall {
					t.Errorf("expected '%s', got '%s'", failures.NoSuchCall, cause)
				}
			})
		})
	})

	t.Run("when no guid is provided", func(t *testing.T) {
		logService.Reset()
		input := core.
			NewInput(services)

		result := LoadCall(input)

		t.Run("the op logs an error", func(t *testing.T) {
			if !logService.ReceivedError(logScope, "no call guid provided") {
				t.Errorf("expected a logged error regarding the missing guid")
			}
		})

		t.Run("the op is a failure", func(t *testing.T) {
			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.NoSuchCall {
				t.Errorf("expected '%s', got '%s'", failures.NoSuchCall, cause)
			}
		})
	})

}
