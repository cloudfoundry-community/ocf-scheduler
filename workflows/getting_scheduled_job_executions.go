package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var GettingScheduledJobExecutions = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadJob,
	ops.LoadExecutions,
)
