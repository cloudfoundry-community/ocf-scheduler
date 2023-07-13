package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var GettingJobExecutions = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadJob,
	ops.LoadExecutionCollection,
)
