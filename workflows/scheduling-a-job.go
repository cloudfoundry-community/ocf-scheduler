package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var SchedulingAJob = dry.NewTransaction(
	ops.VerifyAuth, //tested
	ops.LoadJob,    //tested
	ops.ValidateScheduleExpression,
	ops.PersistSchedule,
	ops.ScheduleJob,
)
