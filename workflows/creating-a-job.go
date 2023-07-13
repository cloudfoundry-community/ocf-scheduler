package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var CreatingAJob = dry.NewTransaction(
	ops.VerifyAuth,
	ops.ValidateAppGUID,
	ops.QuerySpace,
	ops.ValidateJobName,
	ops.ValidateJobCommand,
	ops.PersistJob,
)
