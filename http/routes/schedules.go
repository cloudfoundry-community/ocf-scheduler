package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func Schedules(e *echo.Echo, services *core.Services) {
	ValidateSchedule(e, services)
}
