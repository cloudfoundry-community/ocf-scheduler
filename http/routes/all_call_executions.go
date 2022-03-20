package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/auth"
)

func AllCallExecutions(e *echo.Echo, services *core.Services) {
	// Get all execution histories for a Call
	// GET /calls/{callGuid}/history
	e.GET("/calls/:guid/history", func(c echo.Context) error {
		if auth.Verify(c) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		return c.JSON(
			http.StatusOK,
			"GET RESULT",
		)
	})
}
