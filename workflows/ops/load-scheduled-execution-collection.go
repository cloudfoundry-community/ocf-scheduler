package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func LoadScheduledExecutionCollection(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	schedule := input.Schedule

	input.ExecutionCollection = input.Services.Executions.BySchedule(
		schedule,
	)

	return dry.Success(input)
}
