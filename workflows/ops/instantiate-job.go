package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func InstantiateJob(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	job := &core.Job{}

	if err := input.Context.Bind(&input); err != nil {
		return dry.Failure("could-not-bind-input")
	}

	input.Executable = job

	return dry.Success(input)
}
