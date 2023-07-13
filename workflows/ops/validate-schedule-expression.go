package ops

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
)

func ValidateScheduleExpression(raw dry.Value) dry.Result {
	tag := "ops.validate-schedule-expression"
	input := core.Inputify(raw)
	logger := input.Services.Logger
	schedule := input.Schedule

	if input.Services.Cron.Validate(schedule.Expression) != nil {
		logger.Error(
			tag,
			"schedule cron expression invalid",
		)

		return dry.Failure(failures.ScheduleExpressionInvalid)
	}

	return dry.Success(input)
}
