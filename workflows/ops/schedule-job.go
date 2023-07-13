package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func ScheduleJob(raw dry.Value) dry.Result {
	input := core.Inputify(raw)
	services := input.Services
	job, _ := input.Executable.ToJob()
	schedule := input.Schedule

	services.Cron.Add(core.NewJobRun(job, schedule, services))

	return dry.Success(input)
}
