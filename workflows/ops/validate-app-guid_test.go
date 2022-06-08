package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_ValidateAppGUID(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	logScope := "ops.validate-app-guid"
	logService := mock.NewLogService()
	services := &core.Services{Logger: logService}

	t.Run("when the appGUID is populated", func(t *testing.T) {
		input := core.NewInput(services).
			WithAppGUID("IGotANeutronFreeOfCharge")

		result := ValidateAppGUID(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})
	})

	t.Run("when the appGUID is empty", func(t *testing.T) {
		input := core.NewInput(services).
			WithAppGUID("")

		result := ValidateAppGUID(input)

		t.Run("the op is a failure", func(t *testing.T) {
			if result.Success() {
				t.Errorf("expected a failure, got a success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.NoAppGUID {
				t.Errorf("expected %s, got %s", failures.NoAppGUID, cause)
			}
		})

		t.Run("an error is logged about the empty appGUID", func(t *testing.T) {
			if !logService.ReceivedError(logScope, "app GUID cannot be blank") {

				t.Errorf("expected the empty AppGUID to be logged")
			}
		})
	})
}
