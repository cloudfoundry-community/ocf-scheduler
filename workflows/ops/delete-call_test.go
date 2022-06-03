package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_DeleteCall(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	callService := mock.NewCallService()
	services := &core.Services{Calls: callService}
	guid, _ := core.GenGUID()
	call := dummyCall(&core.Call{GUID: guid, Name: "delete-call-test"})

	t.Run("when the provided executable is a call", func(t *testing.T) {
		input := core.
			NewInput(services).
			WithExecutable(call)

		t.Run("when deletion succeeds", func(t *testing.T) {
			if _, err := callService.Persist(call); err != nil {
				t.Errorf("could not persist call: %s", err.Error())
			}

			t.Run("the op is a success", func(t *testing.T) {
				result := DeleteCall(input)
				if result.Failure() {
					failure := core.Causify(result.Error())

					t.Errorf("expected a success, got failure '%s'", failure)
				}
			})

		})

		t.Run("when deletion fails", func(t *testing.T) {
			// okay, this is dirty, but we need an easy way to test a deletion
			// failure in the mock call service, which always returns clean, so
			// we added a super sad scenario where it always returns error
			call.Name = "sad-face"
			input.Executable = call

			t.Run("the op is a failure", func(t *testing.T) {
				result := DeleteCall(input)
				if result.Success() {
					t.Errorf("expected a failure, got success")
				}
			})
		})
	})

	t.Run("when the provided executable is a job", func(t *testing.T) {
		input := core.
			NewInput(services).
			WithExecutable(dummyJob(nil))

		t.Run("the op fails due to a type mismatch", func(t *testing.T) {
			result := DeleteCall(input)

			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.ExecutableTypeMismatch {
				t.Errorf("expected an executable type mismatch, got %s", cause)
			}
		})
	})

	t.Run("when the provided executable is nil", func(t *testing.T) {
		input := core.
			NewInput(services)

		t.Run("the op fails due to a type mismatch", func(t *testing.T) {
			result := DeleteCall(input)

			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.ExecutableTypeMismatch {
				t.Errorf("expected an executable type mismatch, got %s", cause)
			}
		})
	})
}
