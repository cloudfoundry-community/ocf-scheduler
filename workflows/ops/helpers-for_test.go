package ops

import (
	"time"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func blank(candidate string) bool {
	return len(candidate) == 0
}

func dummyCall(call *core.Call) *core.Call {
	if call == nil {
		call = &core.Call{}
	}

	now := time.Now().UTC()
	call.CreatedAt = now
	call.UpdatedAt = now

	if blank(call.GUID) {
		call.GUID, _ = core.GenGUID()
	}

	if blank(call.Name) {
		call.Name = "dummy-call-" + call.GUID
	}

	if blank(call.URL) {
		call.URL = "http://example.com"
	}

	if blank(call.AuthHeader) {
		call.AuthHeader = "dummy"
	}

	if blank(call.AppGUID) {
		call.AppGUID, _ = core.GenGUID()
	}

	if blank(call.SpaceGUID) {
		call.SpaceGUID, _ = core.GenGUID()
	}

	return call
}

func dummyJob(job *core.Job) *core.Job {
	if job == nil {
		job = &core.Job{}
	}

	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	if blank(job.GUID) {
		job.GUID, _ = core.GenGUID()
	}

	if blank(job.Name) {
		job.Name = "dummy-job-" + job.GUID
	}

	if blank(job.Command) {
		job.Command = "echo 'I sure am a dummy job'"
	}

	if blank(job.AppGUID) {
		job.AppGUID, _ = core.GenGUID()
	}

	if blank(job.SpaceGUID) {
		job.SpaceGUID, _ = core.GenGUID()
	}

	return job
}

func dummySchedule(schedule *core.Schedule) *core.Schedule {
	if schedule == nil {
		schedule = &core.Schedule{}
	}

	now := time.Now().UTC()
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	if blank(schedule.GUID) {
		schedule.GUID, _ = core.GenGUID()
	}

	if blank(schedule.Expression) {
		schedule.Expression = "* * * * *"
	}

	if blank(schedule.ExpressionType) {
		schedule.ExpressionType = "cron_expression"
	}

	if blank(schedule.RefGUID) {
		schedule.RefGUID, _ = core.GenGUID()
	}

	if blank(schedule.RefType) {
		schedule.RefType = "job"
	}

	return schedule
}

func dummyExecution(entity *core.Execution) *core.Execution {
	if entity == nil {
		entity = &core.Execution{}
	}

	now := time.Now().UTC()
	entity.ScheduledTime = now
	entity.ExecutionStartTime = now
	entity.ExecutionEndTime = now

	if blank(entity.GUID) {
		entity.GUID, _ = core.GenGUID()
	}

	if blank(entity.TaskGUID) {
		entity.TaskGUID, _ = core.GenGUID()
	}

	if blank(entity.Message) {
		entity.Message = "o hi"
	}

	if blank(entity.State) {
		entity.State = "Minnesota"
	}

	if blank(entity.RefType) {
		entity.RefType = "job"
	}

	if blank(entity.RefGUID) {
		entity.RefGUID, _ = core.GenGUID()
	}

	return entity
}
