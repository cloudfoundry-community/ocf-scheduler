package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func CreateJob(e *echo.Echo, services *core.Services) {
	// Create Job
	// POST /jobs?app_guid=string
	e.POST("/jobs", func(c echo.Context) error {
		tag := "create-job"

		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			services.Logger.Error(tag, "authentication to this endpoint failed")
			return c.JSON(http.StatusUnauthorized, "")
		}

		appGUID := c.QueryParam("app_guid")

		input := &core.Job{}

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

		if len(input.Name) == 0 {
			services.Logger.Error(tag, "got a blank job name")
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		if len(input.Command) == 0 {
			services.Logger.Error(tag, "got a blank job command")
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		job, err := services.Jobs.Persist(input)
		if err != nil {
			services.Logger.Error(tag, "could not persist the job")
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		success := fmt.Sprintf(
			"successfully created job %s for app %s",
			job.Name,
			appGUID,
		)

		services.Logger.Info(tag, success)
		return c.JSON(
			http.StatusCreated,
			job,
		)
	})
}
