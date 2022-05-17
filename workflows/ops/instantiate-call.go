package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func InstantiateCall(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	call := &core.Call{}

	if err := input.Context.Bind(&call); err != nil {
		return dry.Failure("could-not-bind-input")
	}

	input.Services.Logger.Info("ops.instantiate-call", fmt.Sprintf("got job %s (%s : %s)", call.Name, call.URL, call.AuthHeader))

	input.Executable = call

	return dry.Success(input)
}
