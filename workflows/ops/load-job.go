package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func LoadJob(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	job, err := input.Services.Jobs.Get(input.Context.Param("guid"))
	if err != nil {
		return dry.Failure("no-such-job")
	}

	input.Executable = job

	return dry.Success(input)
}
