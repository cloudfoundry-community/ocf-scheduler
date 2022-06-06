package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_VerifyAuth(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	authService := mock.NewAuthService()
	logService := mock.NewLogService()
	services := &core.Services{Auth: authService, Logger: logService}
	goodAuth := "jeremy"

	t.Run("when the auth info is provided", func(t *testing.T) {
		input := core.
			NewInput(services).
			WithAuth(goodAuth)

		t.Run("and the user is authorized", func(t *testing.T) {
			t.Run("the op is a success", func(t *testing.T) {
				result := VerifyAuth(input)
				if result.Failure() {
					failure := core.Causify(result.Error())

					t.Errorf("expected a success, got failure '%s'", failure)
				}
			})
		})

		t.Run("but the user is not authorized", func(t *testing.T) {
			input = input.WithAuth("bad auth")

			t.Run("the op is a failure", func(t *testing.T) {
				result := VerifyAuth(input)
				if result.Success() {
					t.Errorf("expected a failure, got success")
				}
			})
		})

	})

	t.Run("when no auth info is provided", func(t *testing.T) {
		input := core.
			NewInput(services)

		logScope := "ops.verify-auth"
		result := VerifyAuth(input)

		t.Run("the op fails", func(t *testing.T) {
			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.AuthFailure {
				t.Errorf("expected an auth failure, got %s", cause)
			}
		})

		t.Run("the auth failure is logged", func(t *testing.T) {
			if !logService.ReceivedError(logScope, "authentication to this endpoint failed") {
				t.Errorf("expected the auth failure to be logged")
			}
		})
	})

}
