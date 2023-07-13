package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
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

		return dry.Failure(failures.JobNameBlank)
	}

	if input.Services.Jobs.Exists(input.Data["appGUID"], job.Name) {
		logger.Error(
			tag,
			"there is already a job by that name for this app",
		)
		return dry.Failure(failures.JobNameNotUnique)
	}

	return dry.Success(input)
}
