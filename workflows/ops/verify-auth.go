package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func VerifyAuth(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	if input.Services.Auth.Verify(input.Auth) != nil {
		input.Services.Logger.Error(
			"ops.verify-auth",
			"authentication to this endpoint failed",
		)

		return dry.Failure(failures.AuthFailure)
	}

	return dry.Success(input)
}
