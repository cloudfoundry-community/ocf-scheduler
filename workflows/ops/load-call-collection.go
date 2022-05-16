package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func LoadCallCollection(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	spaceGUID := input.Context.QueryParam("space_guid")
	if len(spaceGUID) > 0 {
		input.CallCollection = input.Services.Calls.InSpace(spaceGUID)
	}

	return dry.Success(input)
}
