package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var CreatingACall = dry.NewTransaction(
	ops.VerifyAuth,
	ops.ValidateAppGUID,
	ops.QuerySpace,
	ops.ValidateCallName,
	ops.ValidateCallURL,
	ops.ValidateCallAuthHeader,
	ops.PersistCall,
)
