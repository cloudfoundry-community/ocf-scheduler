package ops

import (
	"strings"
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_QuerySpace(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	logScope := "ops.query-space"
	infoService := mock.NewInfoService()
	logService := mock.NewLogService()
	services := &core.Services{Info: infoService, Logger: logService}
	appGUID, _ := core.GenGUID()

	t.Run("when a valid app GUID is provided", func(t *testing.T) {
		input := core.NewInput(services).
			WithAppGUID(appGUID)

		result := QuerySpace(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting payload contains a populated space GUID", func(t *testing.T) {
			payload := core.Inputify(result.Value())

			if payload.Data["spaceGUID"] == "" {
				t.Errorf("expected a populated space GUID")
			}
		})
	})

	t.Run("when a bad app GUID is provided", func(t *testing.T) {
		input := core.NewInput(services).
			WithAppGUID("sad-face")

		result := QuerySpace(input)

		t.Run("the op is a failure", func(t *testing.T) {
			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.NoSpaceGUID {
				t.Errorf("expected %s, got %s", failures.NoSpaceGUID, cause)
			}
		})

		t.Run("an error is logged regarding the space lookup failure", func(t *testing.T) {
			found := false
			for _, line := range logService.ErrorsFor(logScope) {
				if strings.HasPrefix(line, "could not get space GUID for app sad-face") {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected the space lookup failure to be logged")
			}
		})
	})

	t.Run("when a blank app GUID is provided", func(t *testing.T) {
		input := core.NewInput(services).
			WithAppGUID("")

		result := QuerySpace(input)

		t.Run("the op is a failure", func(t *testing.T) {
			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.NoAppGUID {
				t.Errorf("expected %s, got %s", failures.NoAppGUID, cause)
			}
		})

		t.Run("an error is logged regarding the blank app GUID", func(t *testing.T) {
			if !logService.ReceivedError(logScope, "an app GUID is required for this operation") {
				t.Errorf("expected the missing app GUID to be logged")
			}
		})
	})

}
