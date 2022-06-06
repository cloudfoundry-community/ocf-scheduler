package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var CreatingAJob = dry.NewTransaction(
	ops.VerifyAuth, //tested
	ops.ValidateAppGUID,
	ops.QuerySpace,
	ops.ValidateJobName,
	ops.ValidateJobCommand,
	ops.PersistJob,
)
