package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
)

func LoadCall(raw dry.Value) dry.Result {
	tag := "ops.load-call"
	input := core.Inputify(raw)
	guid := input.Data["guid"]

	if guid == "" {
		input.Services.Logger.Error(
			tag,
			"no call guid provided",
		)
	}

	call, err := input.Services.Calls.Get(guid)
	if err != nil {
		input.Services.Logger.Error(
			tag,
			fmt.Sprintf("could not find call with guid %s", guid),
		)

		return dry.Failure(failures.NoSuchCall)
	}

	input.Executable = call

	return dry.Success(input)
}
