package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var GettingScheduledCallExecutions = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadCall,
	ops.LoadSchedule,
	ops.LoadScheduledExecutionCollection,
)
