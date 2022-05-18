package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var ExecutingAJob = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadJob,
	ops.InstantiateExecution,
	ops.PersistExecution,
	ops.ExecuteJob,
)
