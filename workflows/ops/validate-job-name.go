package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func ValidateJobName(raw dry.Value) dry.Result {
	tag := "ops.validate-job-name"
	input := core.Inputify(raw)
	logger := input.Services.Logger

	job, _ := input.Executable.ToJob()

	if job.Name == "" {
		logger.Error(
			tag,
			"job name cannot be blank",
		)

		return dry.Failure("job-name-blank")
	}

	if input.Services.Jobs.Exists(input.Data["appGUID"], job.Name) {
		logger.Error(
			tag,
			"there is already a job by that name for this app",
		)
		return dry.Failure("job-name-not-unique-for-app")
	}

	return dry.Success(input)
}
