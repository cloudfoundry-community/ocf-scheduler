package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func CreateCall(e *echo.Echo, services *core.Services) {
	// Create Call
	// POST /calls?app_guid=string
	e.POST("/calls", func(c echo.Context) error {
		tag := "create-call"

		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			services.Logger.Error(tag, "authentication to this endpoint failed")
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
			services.Logger.Error(tag, "got a blank call name")
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		if len(input.URL) == 0 {
			services.Logger.Error(tag, "got a blank call URL")
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		if len(input.AuthHeader) == 0 {
			services.Logger.Error(tag, "got a blank call auth header")
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		call, err := services.Calls.Persist(input)
		if err != nil {
			services.Logger.Error(tag, "could not persist the call")
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		success := fmt.Sprintf(
			"successfully created call %s for app %s",
			call.Name,
			appGUID,
		)

		services.Logger.Info(tag, success)
		return c.JSON(
			http.StatusCreated,
			call,
		)
	})
}
