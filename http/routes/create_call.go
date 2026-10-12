package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
)

func CreateCall(e *echo.Echo, services *core.Services) {
	// Create Call
	// POST /calls?app_guid=string
	e.POST("/calls", func(c echo.Context) error {
		tag := "create-call"

		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		appGUID := c.QueryParam("app_guid")
		if len(appGUID) == 0 {
			return c.JSON(missing(services, tag, auth, http.StatusUnprocessableEntity), "")
		}

		spaceGUID, err := services.Info.GetSpaceGUIDForApp(appGUID)
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("could not get space GUID for app %s", appGUID))
			return c.JSON(missing(services, tag, auth, http.StatusUnprocessableEntity), "")
		}

		if status := authorize(services, tag, auth, spaceGUID, http.StatusForbidden); status != 0 {
			return c.JSON(status, "")
		}

		status, body, admin := requireBinding(services, tag, auth, appGUID)
		if status != 0 {
			return c.JSON(status, body)
		}

		input := &core.Call{}

		if err := c.Bind(&input); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		input.AppGUID = appGUID
		input.SpaceGUID = spaceGUID
		services.Logger.Debug(tag, fmt.Sprintf("Space GUID is '%s'", spaceGUID))

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

		var call *core.Call
		status, body = whileBound(services, tag, appGUID, admin, func() (int, any) {
			call, err = services.Calls.Persist(input)
			if err != nil {
				services.Logger.Error(tag, "could not persist the call")
				return http.StatusUnprocessableEntity, ""
			}

			return 0, nil
		})
		if status != 0 {
			return c.JSON(status, body)
		}

		success := fmt.Sprintf(
			"successfully created call %s for app %s",
			call.Name,
			appGUID,
		)

		services.Logger.Info(tag, success)
		return c.JSON(
			http.StatusCreated,
			presenters.AsCall(call),
		)
	})
}
