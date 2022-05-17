package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func ValidateJobCommand(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	job, _ := input.Executable.ToJob()

	if job.Command == "" {
		input.Services.Logger.Error(
			"ops.validate-job-command",
			"job command cannot be blank",
		)

		return dry.Failure("job-command-blank")
	}

	return dry.Success(input)
}
