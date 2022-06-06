package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func LoadScheduleCollection(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	executable := input.Executable

	if executable == nil {
		return dry.Failure(failures.ExecutableTypeMismatch)
	}

	input.ScheduleCollection = input.Services.Schedules.ByRef(
		executable.RefType(),
		executable.RefGUID(),
	)

	return dry.Success(input)
}
