package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var GettingAJob = dry.NewTransaction(
	ops.VerifyAuth, //tested
	ops.LoadJob,    //tested
)
