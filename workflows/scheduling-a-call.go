package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var SchedulingACall = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadCall,
	ops.ValidateScheduleExpression,
	ops.PersistSchedule,
	ops.ScheduleCall,
)
