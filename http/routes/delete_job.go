package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

func DeleteJob(e *echo.Echo, services *core.Services) {
	// Delete a Job
	// DELETE /jobs/{jobGuid}
	e.DELETE("/jobs/:guid", func(c echo.Context) error {
		tag := "delete-job"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			services.Logger.Error(tag, "authentication failed")
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		// look up the job
		job, err := services.Jobs.Get(guid)
		if err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("job %s not found", guid))
			return c.JSON(
				http.StatusNotFound,
				"",
			)
		}

		// delete things associated with the job
		for _, schedule := range services.Schedules.ByJob(job) {
			err = workflows.DeletingASchedule(services, schedule, job)
			if err != nil {
				services.Logger.Error(tag, fmt.Sprintf("failed to delete schedule %s for job %s: %v", schedule.GUID, guid, err))
				return c.JSON(http.StatusInternalServerError, "")
			}
		}

		// actually delete the job
		err = services.Jobs.Delete(job)
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to delete job %s: %v", guid, err))
			return c.JSON(
				http.StatusInternalServerError,
				"",
			)
		}

		services.Logger.Info(tag, fmt.Sprintf("deleted job %s", guid))
		return c.JSON(
			http.StatusNoContent,
			"",
		)
	})
}
