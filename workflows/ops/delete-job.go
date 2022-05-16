package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func DeleteJob(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	job, err := input.Executable.ToJob()
	if err != nil {
		return dry.Failure("executable-type-mismatch")
	}

	err = input.Services.Jobs.Delete(job)
	if err != nil {
		return dry.Failure("delete-job-failed")
	}

	return dry.Success(input)
}
