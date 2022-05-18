package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func CreateCall(e *echo.Echo, services *core.Services) {
	// Create Call
	// POST /calls?app_guid=string
	e.POST("/calls", func(c echo.Context) error {
		result := workflows.
			CreatingACall.
			Call(core.NewInput(c, services))

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case "auth-failure":
				return c.JSON(http.StatusUnauthorized, "")
			default:
				return c.JSON(http.StatusUnprocessableEntity, "")
			}
		}

		call, _ := core.Inputify(result.Value()).Executable.ToCall()

		return c.JSON(
			http.StatusCreated,
			call,
		)
	})
}
