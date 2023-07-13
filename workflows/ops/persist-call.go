package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
)

func PersistCall(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	services := input.Services

	candidate, _ := input.Executable.ToCall()
	candidate.AppGUID = input.Data["appGUID"]
	candidate.SpaceGUID = input.Data["spaceGUID"]

	call, err := services.Calls.Persist(candidate)
	if err != nil {
		services.Logger.Error(
			"ops.persist-call",
			fmt.Sprintf("could not persist the call: %s", err.Error()),
		)

		return dry.Failure(failures.PersistCallFailure)
	}

	input.Executable = call

	return dry.Success(input)
}
