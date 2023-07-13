package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var UnschedulingACall = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadCall,
	ops.LoadSchedule,
	ops.DeleteSchedule,
)
