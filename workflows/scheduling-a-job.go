package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var SchedulingAJob = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadJob,
	ops.ValidateScheduleExpression,
	ops.PersistSchedule,
	ops.ScheduleJob,
)
