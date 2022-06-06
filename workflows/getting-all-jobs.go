package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var GettingAllJobs = dry.NewTransaction(
	ops.VerifyAuth,        //tested
	ops.LoadJobCollection, //tested
)
