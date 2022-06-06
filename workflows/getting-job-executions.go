package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var GettingJobExecutions = dry.NewTransaction(
	ops.VerifyAuth,              //tested
	ops.LoadJob,                 //tested
	ops.LoadExecutionCollection, //tested
)
