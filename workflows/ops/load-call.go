package ops

import (
	"fmt"

	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
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

		return dry.Failure(failures.NoSuchCall)
	}

	input.Executable = call

	return dry.Success(input)
}
