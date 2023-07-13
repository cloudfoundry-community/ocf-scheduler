package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var ExecutingACall = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadCall,
	ops.PersistExecution,
	ops.ExecuteCall,
)
