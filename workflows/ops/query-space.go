package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
)

func QuerySpace(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	appGUID := input.Data["appGUID"]

	spaceGUID, err := input.Services.Info.GetSpaceGUIDForApp(appGUID)
	if err != nil || spaceGUID == "" {
		input.Services.Logger.Error(
			"ops.query-space",
			fmt.Sprintf(
				"could not get space GUId for app %s: %s",
				appGUID,
				err.Error(),
			),
		)

		return dry.Failure(failures.NoSpaceGUID)
	}

	input.Data["spaceGUID"] = spaceGUID

	return dry.Success(input)
}
