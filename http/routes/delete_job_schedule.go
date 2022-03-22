package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/auth"
)

func DeleteJobSchedule(e *echo.Echo, services *core.Services) {
	// Delete the given schedule for the given Job
	// DELETE /jobs/{jobGuid}/schedules/{scheduleGuid}
	e.DELETE("/jobs/:guid/schedules/:schedule_guid", func(c echo.Context) error {
		if auth.Verify(c) != nil {
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

		run := &core.Run{Job: job, Schedule: schedule, Services: services}

		if services.Cron.Delete(run) != nil {
			return c.JSON(http.StatusInternalServerError, "")
		}

		if services.Schedules.Delete(schedule) != nil {
			return c.JSON(http.StatusInternalServerError, "")
		}

		return c.JSON(
			http.StatusNoContent,
			"",
		)
	})
}
