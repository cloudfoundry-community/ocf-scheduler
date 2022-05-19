package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func ExecuteCall(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	services := input.Services
	call, _ := input.Executable.ToCall()
	execution := input.Execution

	services.Runner.Execute(services, execution, call)

	return dry.Success(input)
}
