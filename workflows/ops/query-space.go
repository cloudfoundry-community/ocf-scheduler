package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func QuerySpace(raw dry.Value) dry.Result {
	tag := "ops.query-space"
	input := core.Inputify(raw)

	appGUID := input.Data["appGUID"]

	if appGUID == "" {
		input.Services.Logger.Error(
			tag,
			"an app GUID is required for this operation",
		)

		return dry.Failure(failures.NoAppGUID)
	}

	spaceGUID, err := input.Services.Info.GetSpaceGUIDForApp(appGUID)
	if err != nil || spaceGUID == "" {
		input.Services.Logger.Error(
			tag,
			fmt.Sprintf(
				"could not get space GUID for app %s: %s",
				appGUID,
				err.Error(),
			),
		)

		return dry.Failure(failures.NoSpaceGUID)
	}

	input.Data["spaceGUID"] = spaceGUID

	return dry.Success(input)
}
