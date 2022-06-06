package workflows

import (
	"github.com/ess/dry"

	"github.com/starkandwayne/scheduler-for-ocf/workflows/ops"
)

var GettingAllCalls = dry.NewTransaction(
	ops.VerifyAuth,         //tested
	ops.LoadCallCollection, //tested
)
