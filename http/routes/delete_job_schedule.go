package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

func DeleteJobSchedule(e *echo.Echo, services *core.Services) {
	// Delete the given schedule for the given Job
	// DELETE /jobs/{jobGuid}/schedules/{scheduleGuid}
	e.DELETE("/jobs/:guid/schedules/:schedule_guid", func(c echo.Context) error {
		tag := "delete-job-schedule"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			services.Logger.Error(tag, "authentication failed")
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		job, err := services.Jobs.Get(guid)
		if err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("job %s not found", guid))
			return c.JSON(http.StatusNotFound, "")
		}

		scheduleGUID := c.Param("schedule_guid")
		schedule, err := services.Schedules.Get(scheduleGUID)
		if err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("schedule %s not found for job %s", scheduleGUID, guid))
			return c.JSON(http.StatusNotFound, "")
		}

		err = workflows.DeletingASchedule(services, schedule, job)
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to delete schedule %s for job %s: %v", scheduleGUID, guid, err))
			return c.JSON(http.StatusInternalServerError, "")
		}

		services.Logger.Info(tag, fmt.Sprintf("deleted schedule %s for job %s", scheduleGUID, guid))
		return c.JSON(
			http.StatusNoContent,
			"",
		)
	})
}
