package ops

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func LoadCall(raw dry.Value) dry.Result {
	input := core.Inputify(raw)

	call, err := input.Services.Calls.Get(input.Context.Param("guid"))
	if err != nil {
		return dry.Failure("no-such-call")
	}

	input.Executable = call

	return dry.Success(input)
}
