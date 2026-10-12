package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
)

func GetCall(e *echo.Echo, services *core.Services) {
	// Get a Call
	// GET /calls/{callGuid}
	e.GET("/calls/:guid", func(c echo.Context) error {
		tag := "get-call"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		guid := c.Param("guid")

		call, err := services.Calls.Get(guid)
		if err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("call %s not found", guid))
			return c.JSON(missing(services, tag, auth, http.StatusNotFound), "")
		}

		if status := authorize(services, tag, auth, call.SpaceGUID, http.StatusNotFound); status != 0 {
			return c.JSON(status, "")
		}

		return c.JSON(
			http.StatusOK,
			presenters.AsCall(call),
		)
	})
}
