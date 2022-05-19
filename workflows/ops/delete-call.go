package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func DeleteCall(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	call, err := input.Executable.ToCall()
	if err != nil {
		return dry.Failure(failures.ExecutableTypeMismatch)
	}

	err = input.Services.Calls.Delete(call)
	if err != nil {
		return dry.Failure(failures.DeleteCallFailed)
	}

	return dry.Success(input)
}
