package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func GetCall(e *echo.Echo, services *core.Services) {
	// Get a Call
	// GET /calls/{callGuid}
	e.GET("/calls/:guid", func(c echo.Context) error {
		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithGUID(c.Param("guid"))

		result := workflows.
			GettingACall.
			Call(input)

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case failures.AuthFailure:
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
