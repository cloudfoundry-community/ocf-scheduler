package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func PersistExecution(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	services := input.Services

	execution, err := services.Executions.Persist(input.Execution)
	if err != nil {
		services.Logger.Error(
			"ops.persist-execution",
			fmt.Sprintf("could not persist the execution: %s", err.Error()),
		)

		return dry.Failure(failures.PersistExecutionFailure)
	}

	input.Execution = execution

	return dry.Success(input)
}
