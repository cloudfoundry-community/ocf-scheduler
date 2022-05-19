package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func VerifyAuth(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	if input.Services.Auth.Verify(input.Data["auth"]) != nil {
		input.Services.Logger.Error(
			"ops.verify-auth",
			"authentication to this endpoint failed",
		)

		return dry.Failure("auth-failure")
	}

	return dry.Success(input)
}
