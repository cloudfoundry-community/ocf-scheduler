package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var GettingCallExecutions = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadCall,
	ops.LoadExecutionCollection,
)
