package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func LoadScheduleCollection(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	executable := input.Executable

	input.ScheduleCollection = input.Services.Schedules.ByRef(
		executable.RefType(),
		executable.RefGUID(),
	)

	return dry.Success(input)
}
