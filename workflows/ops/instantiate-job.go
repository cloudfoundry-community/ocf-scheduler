package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func InstantiateJob(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	job := &core.Job{}

	if err := input.Context.Bind(&job); err != nil {
		return dry.Failure("could-not-bind-input")
	}

	input.Services.Logger.Info("ops.instantiate-job", fmt.Sprintf("got job %s (%s)", job.Name, job.Command))

	input.Executable = job

	return dry.Success(input)
}
