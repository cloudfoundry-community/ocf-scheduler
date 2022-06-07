package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var UnschedulingAJob = dry.NewTransaction(
	ops.VerifyAuth,   //tested
	ops.LoadJob,      //tested
	ops.LoadSchedule, //tested
	ops.DeleteSchedule,
)
