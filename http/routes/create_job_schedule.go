package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/auth"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
)

func CreateJobSchedule(e *echo.Echo, services *core.Services) {
	// Schedule a Job to run later
	// POST /jobs/{jobGuid}/schedules
	e.POST("/jobs/:guid/schedules", func(c echo.Context) error {
		if auth.Verify(c) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		_, err := services.Jobs.Get(guid)
		if err != nil {
			return c.JSON(
				http.StatusNotFound,
				"",
			)
		}

		input := &core.Schedule{}

		if err = c.Bind(&input); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		input.RefGUID = guid
		input.RefType = "job"

		schedule, err := services.Schedules.Persist(input)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		return c.JSON(
			http.StatusCreated,
			presenters.AsJobSchedule(schedule),
		)
	})
}
