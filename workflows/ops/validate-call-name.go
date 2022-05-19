package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func ValidateCallName(raw dry.Value) dry.Result {
	tag := "ops.validate-call-name"
	input := core.Inputify(raw)
	logger := input.Services.Logger

	call, _ := input.Executable.ToCall()

	if call.Name == "" {
		logger.Error(
			tag,
			"call name cannot be blank",
		)

		return dry.Failure(failures.CallNameBlank)
	}

	if input.Services.Calls.Exists(input.Data["appGUID"], call.Name) {
		logger.Error(
			tag,
			"there is already a call by that name for this app",
		)
		return dry.Failure(failures.CallNameNotUnique)
	}

	return dry.Success(input)
}
