package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var GettingCallExecutions = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadCall,
	ops.LoadExecutionCollection,
)
