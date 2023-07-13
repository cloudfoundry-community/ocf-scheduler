package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func Apply(e *echo.Echo, services *core.Services) {
	Jobs(e, services)
	Calls(e, services)
}
