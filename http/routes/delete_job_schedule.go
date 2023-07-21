package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

func DeleteJobSchedule(e *echo.Echo, services *core.Services) {
	// Delete the given schedule for the given Job
	// DELETE /jobs/{jobGuid}/schedules/{scheduleGuid}
	e.DELETE("/jobs/:guid/schedules/:schedule_guid", func(c echo.Context) error {
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		job, err := services.Jobs.Get(guid)
		if err != nil {
			return c.JSON(http.StatusNotFound, "")
		}

		scheduleGUID := c.Param("schedule_guid")
		schedule, err := services.Schedules.Get(scheduleGUID)
		if err != nil {
			return c.JSON(http.StatusNotFound, "")
		}

		err = workflows.DeletingASchedule(services, schedule, job)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "")
		}

		return c.JSON(
			http.StatusNoContent,
			"",
		)
	})
}
