package mock

import "github.com/cloudfoundry-community/ocf-scheduler/core"

// The mocks must keep up with the core interfaces.
var (
	_ core.JobService      = &JobService{}
	_ core.CallService     = &CallService{}
	_ core.ScheduleService = &ScheduleService{}
	_ core.AuthService     = &AuthService{}
)
