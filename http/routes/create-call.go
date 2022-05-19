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
		candidate := &core.Call{}
		c.Bind(&candidate)

		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithExecutable(candidate).
			WithAppGUID(c.QueryParam("app_guid"))

		result := workflows.
			CreatingACall.
			Call(input)

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
