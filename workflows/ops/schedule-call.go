package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func ScheduleCall(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	services := input.Services
	call, _ := input.Executable.ToCall()
	schedule := input.Schedule

	services.Cron.Add(core.NewCallRun(call, schedule, services))

	return dry.Success(input)
}
