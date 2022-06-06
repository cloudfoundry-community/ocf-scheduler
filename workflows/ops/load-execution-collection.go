package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func LoadExecutionCollection(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	executable := input.Executable

	if executable == nil {
		return dry.Failure(failures.ExecutableTypeMismatch)
	}

	input.ExecutionCollection = input.Services.Executions.ByRef(
		executable.RefType(),
		executable.RefGUID(),
	)

	return dry.Success(input)
}
