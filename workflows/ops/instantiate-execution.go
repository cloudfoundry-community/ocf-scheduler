package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func InstantiateExecution(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	executable := input.Executable

	execution := &core.Execution{}

	if err := input.Context.Bind(&execution); err != nil {
		return dry.Failure("could-not-bind-input")
	}

	execution.RefGUID = executable.RefGUID()
	execution.RefType = executable.RefType()

	input.Execution = execution

	return dry.Success(input)
}
