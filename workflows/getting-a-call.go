package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var GettingACall = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadCall,
)
