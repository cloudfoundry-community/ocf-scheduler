package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var UnschedulingAJob = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadJob,
	ops.LoadSchedule,
	ops.DeleteSchedule,
)
