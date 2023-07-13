package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
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
