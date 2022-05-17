package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func ValidateCallURL(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	call, _ := input.Executable.ToCall()

	if call.URL == "" {
		input.Services.Logger.Error(
			"ops.validate-call-url",
			"call url cannot be blank",
		)

		return dry.Failure("call-url-blank")
	}

	return dry.Success(input)
}
