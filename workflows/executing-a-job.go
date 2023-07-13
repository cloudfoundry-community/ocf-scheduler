package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var ExecutingAJob = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadJob,
	ops.PersistExecution,
	ops.ExecuteJob,
)
