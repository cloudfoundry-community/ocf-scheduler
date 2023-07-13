package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
)

func DeleteJob(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	job, err := input.Executable.ToJob()
	if err != nil {
		return dry.Failure(failures.ExecutableTypeMismatch)
	}

	err = input.Services.Jobs.Delete(job)
	if err != nil {
		return dry.Failure(failures.DeleteJobFailed)
	}

	return dry.Success(input)
}
