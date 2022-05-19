package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func ValidateJobCommand(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	job, _ := input.Executable.ToJob()

	if job.Command == "" {
		input.Services.Logger.Error(
			"ops.validate-job-command",
			"job command cannot be blank",
		)

		return dry.Failure(failures.JobCommandBlank)
	}

	return dry.Success(input)
}
