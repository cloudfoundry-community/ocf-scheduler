package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func ValidateCallAuthHeader(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	call, _ := input.Executable.ToCall()

	if call.URL == "" {
		input.Services.Logger.Error(
			"ops.validate-call-auth-header",
			"call auth header cannot be blank",
		)

		return dry.Failure(failures.CallAuthHeaderBlank)
	}

	return dry.Success(input)
}
