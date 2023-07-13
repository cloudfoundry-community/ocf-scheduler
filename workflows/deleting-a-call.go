package workflows

import (
	"github.com/ess/dry"

	"github.com/cloudfoundry-community/ocf-scheduler/workflows/ops"
)

var DeletingACall = dry.NewTransaction(
	ops.VerifyAuth,
	ops.LoadCall,
	ops.LoadScheduleCollection,
	ops.DeleteScheduleCollection,
	ops.DeleteCall,
)
