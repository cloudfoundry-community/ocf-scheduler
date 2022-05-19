package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func PersistJob(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	services := input.Services

	candidate, _ := input.Executable.ToJob()
	candidate.AppGUID = input.Data["appGUID"]
	candidate.SpaceGUID = input.Data["spaceGUID"]

	job, err := services.Jobs.Persist(candidate)
	if err != nil {
		services.Logger.Error(
			"ops.persist-job",
			fmt.Sprintf("could not persist the job: %s", err.Error()),
		)

		return dry.Failure(failures.PersistJobFailure)
	}

	input.Executable = job

	return dry.Success(input)
}
