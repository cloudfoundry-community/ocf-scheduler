package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/auth"
)

func CreateJob(e *echo.Echo, services *core.Services) {
	// Create Job
	// POST /jobs?app_guid=string
	e.POST("/jobs", func(c echo.Context) error {
		if auth.Verify(c) != nil {
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

		if len(input.Name) == 0 {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		if len(input.Command) == 0 {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		job, err := services.Jobs.Persist(input)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		return c.JSON(
			http.StatusCreated,
			job,
		)
	})
}
