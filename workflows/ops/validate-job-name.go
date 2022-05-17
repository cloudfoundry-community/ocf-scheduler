package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func ValidateJobName(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	job, _ := input.Executable.ToJob()

	if job.Name == "" {
		input.Services.Logger.Error(
			"ops.validate-job-name",
			"job name cannot be blank",
		)

		return dry.Failure("job-name-blank")
	}

	return dry.Success(input)
}
