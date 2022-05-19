package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func LoadCall(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	guid := input.Data["guid"]

	call, err := input.Services.Calls.Get(guid)
	if err != nil {
		input.Services.Logger.Error(
			"ops.load-call",
			fmt.Sprintf("could not find call with guid %s", guid),
		)

		return dry.Failure("no-such-call")
	}

	input.Executable = call

	return dry.Success(input)
}
