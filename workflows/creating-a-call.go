package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var CreatingACall = dry.NewTransaction(
	ops.VerifyAuth, //tested
	ops.ValidateAppGUID,
	ops.QuerySpace,
	ops.ValidateCallName,
	ops.ValidateCallURL,
	ops.ValidateCallAuthHeader,
	ops.PersistCall,
)
