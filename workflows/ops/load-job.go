package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func LoadJob(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	guid := input.Data["guid"]

	job, err := input.Services.Jobs.Get(guid)
	if err != nil {
		input.Services.Logger.Error(
			"ops.load-job",
			fmt.Sprintf("could not find job with guid %s", guid),
		)

		return dry.Failure(failures.NoSuchJob)
	}

	input.Executable = job

	return dry.Success(input)
}
