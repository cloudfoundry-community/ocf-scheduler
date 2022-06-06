package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var GettingAllJobSchedules = dry.NewTransaction(
	ops.VerifyAuth,             //tested
	ops.LoadJob,                //tested
	ops.LoadScheduleCollection, //tested
)
