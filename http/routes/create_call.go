package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/auth"
)

func CreateCall(e *echo.Echo, services *core.Services) {
	// Create Call
	// POST /calls?app_guid=string
	e.POST("/calls", func(c echo.Context) error {
		if auth.Verify(c) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		appGUID := c.QueryParam("app_guid")

		input := &core.Call{}

		if err := c.Bind(&input); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		input.AppGUID = appGUID
		if len(appGUID) == 0 {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		input.SpaceGUID = services.Environment.SpaceGUID()

		if len(input.Name) == 0 {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		if len(input.URL) == 0 {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		if len(input.AuthHeader) == 0 {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		call, err := services.Calls.Persist(input)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		return c.JSON(
			http.StatusCreated,
			call,
		)
	})
}
