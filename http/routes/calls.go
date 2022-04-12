package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func Calls(e *echo.Echo, services *core.Services) {
	// Pure Call routes
	AllCalls(e, services)
	CreateCall(e, services)
	GetCall(e, services)
	DeleteCall(e, services)

	// Execution-centric subroutes
	ExecuteCall(e, services)
	AllCallExecutions(e, services)
	AllCallScheduleExecutions(e, services)

	// Schedule-centric subroutes
	AllCallSchedules(e, services)
	CreateCallSchedule(e, services)
	DeleteCallSchedule(e, services)
}
