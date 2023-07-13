package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
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

	input.Data["appGUID"] = appGUID

	return dry.Success(input)
}
