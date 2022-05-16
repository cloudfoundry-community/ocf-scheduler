package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func GetCall(e *echo.Echo, services *core.Services) {
	// Get a Call
	// GET /calls/{callGuid}
	e.GET("/calls/:guid", func(c echo.Context) error {
		result := workflows.
			GettingACall.
			Call(core.NewInput(c, services))

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case "auth-failure":
				return c.JSON(http.StatusUnauthorized, "")
			default:
				return c.JSON(http.StatusNotFound, "")
			}
		}

		call, _ := core.Inputify(result.Value()).Executable.ToCall()

		return c.JSON(
			http.StatusOK,
			call,
		)
	})
}
