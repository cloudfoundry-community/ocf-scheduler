package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func LoadJobCollection(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	spaceGUID := input.Context.QueryParam("space_guid")
	if len(spaceGUID) > 0 {
		input.JobCollection = input.Services.Jobs.InSpace(spaceGUID)
	}

	return dry.Success(input)
}
