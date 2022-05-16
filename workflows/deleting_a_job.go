package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var DeletingAJob = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadJob,
	ops.LoadSchedules,
	ops.DeleteScheduleCollection,
	ops.DeleteJob,
)
