package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func LoadExecutionCollection(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	executable := input.Executable

	input.ExecutionCollection = input.Services.Executions.ByRef(
		executable.RefType(),
		executable.RefGUID(),
	)

	return dry.Success(input)
}
