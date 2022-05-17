package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func ValidateCallAuthHeader(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	call, _ := input.Executable.ToCall()

	if call.URL == "" {
		input.Services.Logger.Error(
			"ops.validate-call-auth-header",
			"call auth header cannot be blank",
		)

		return dry.Failure("call-auth-header-blank")
	}

	return dry.Success(input)
}
