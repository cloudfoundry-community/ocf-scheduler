package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
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
