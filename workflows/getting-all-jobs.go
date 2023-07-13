package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var GettingAllJobs = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadJobCollection,
)
