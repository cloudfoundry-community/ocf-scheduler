package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func CreateCall(e *echo.Echo, services *core.Services) {
	// Create Call
	// POST /calls?app_guid=string
	e.POST("/calls", func(c echo.Context) error {
		tag := "create-call"

		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if authErr := services.Auth.Verify(auth); authErr != nil {
			services.Logger.Error(
				tag,
				fmt.Sprintf("authentication to this endpoint failed: %s", authErr.Error()),
			)
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

		spaceGUID, err := services.Info.GetSpaceGUIDForApp(appGUID)
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("could not get space GUID for app %s", appGUID))
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		input.SpaceGUID = spaceGUID
		services.Logger.Info(tag, fmt.Sprintf("Space GUID is '%s'", spaceGUID))

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
