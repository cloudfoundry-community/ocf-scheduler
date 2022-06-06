package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var SchedulingACall = dry.NewTransaction(
	ops.VerifyAuth, //tested
	ops.LoadCall,   //tested
	ops.ValidateScheduleExpression,
	ops.PersistSchedule,
	ops.ScheduleCall,
)
