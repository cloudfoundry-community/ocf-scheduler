package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func LoadJobCollection(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	spaceGUID := input.Data["spaceGUID"]
	if len(spaceGUID) > 0 {
		input.JobCollection = input.Services.Jobs.InSpace(spaceGUID)
	}

	return dry.Success(input)
}
