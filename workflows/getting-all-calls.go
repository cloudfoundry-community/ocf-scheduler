package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var GettingAllCalls = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadCallCollection,
)
