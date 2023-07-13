package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
)

func ValidateCallURL(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	call, _ := input.Executable.ToCall()

	if call.URL == "" {
		input.Services.Logger.Error(
			"ops.validate-call-url",
			"call url cannot be blank",
		)

		return dry.Failure(failures.CallURLBlank)
	}

	return dry.Success(input)
}
