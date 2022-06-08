package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func ValidateAppGUID(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	appGUID := input.Data["appGUID"]
	if appGUID == "" {
		input.Services.Logger.Error(
			"ops.validate-app-guid",
			"app GUID cannot be blank",
		)

		return dry.Failure(failures.NoAppGUID)
	}

	return dry.Success(input)
}
