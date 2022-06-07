package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var UnschedulingACall = dry.NewTransaction(
	ops.VerifyAuth,   //tested
	ops.LoadCall,     //tested
	ops.LoadSchedule, //tested
	ops.DeleteSchedule,
)
