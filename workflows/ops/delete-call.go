package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func DeleteCall(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	call, err := input.Executable.ToCall()
	if err != nil {
		return dry.Failure("executable-type-mismatch")
	}

	err = input.Services.Calls.Delete(call)
	if err != nil {
		return dry.Failure("delete-call-failed")
	}

	return dry.Success(input)
}
