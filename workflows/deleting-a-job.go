package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var DeletingAJob = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadJob,
	ops.LoadScheduleCollection,
	ops.DeleteScheduleCollection,
	ops.DeleteJob,
)
