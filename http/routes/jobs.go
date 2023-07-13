package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func Jobs(e *echo.Echo, services *core.Services) {
	// Pure Job Routes
	AllJobs(e, services)
	CreateJob(e, services)
	GetJob(e, services)
	DeleteJob(e, services)

	// Execution-centric subroutes
	ExecuteJob(e, services)
	AllJobExecutions(e, services)
	AllJobScheduleExecutions(e, services)

	// Schedule-centric subroutes
	AllJobSchedules(e, services)
	CreateJobSchedule(e, services)
	DeleteJobSchedule(e, services)
}
