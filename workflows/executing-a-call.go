package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var ExecutingACall = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadCall,
	ops.PersistExecution,
	ops.ExecuteCall,
)
