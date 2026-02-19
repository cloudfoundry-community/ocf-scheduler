package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
)

func CreateJobSchedule(e *echo.Echo, services *core.Services) {
	// Schedule a Job to run later
	// POST /jobs/{jobGuid}/schedules
	e.POST("/jobs/:guid/schedules", func(c echo.Context) error {
		tag := "create-job-schedule"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			services.Logger.Error(tag, "authentication failed")
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		job, err := services.Jobs.Get(guid)
		if err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("job %s not found", guid))
			return c.JSON(
				http.StatusNotFound,
				"",
			)
		}

		input := &core.Schedule{}

		if err = c.Bind(&input); err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to parse schedule request for job %s: %v", guid, err))
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		input.RefGUID = guid
		input.RefType = "job"

		if err := services.Cron.Validate(input.Expression); err != nil {
			services.Logger.Error(tag, fmt.Sprintf("invalid cron expression '%s' for job %s: %v", input.Expression, guid, err))
			return c.JSON(http.StatusUnprocessableEntity, err.Error())
		}

		schedule, err := services.Schedules.Persist(input)
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to persist schedule for job %s: %v", guid, err))
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		if err := services.Cron.Add(core.NewJobRun(job, schedule, services)); err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to add cron entry for job %s: %v", guid, err))
			return c.JSON(http.StatusUnprocessableEntity, err.Error())
		}

		services.Logger.Info(tag, fmt.Sprintf("created schedule %s for job %s", schedule.GUID, guid))
		return c.JSON(
			http.StatusCreated,
			presenters.AsJobSchedule(schedule),
		)
	})
}
