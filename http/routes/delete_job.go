package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func DeleteJob(e *echo.Echo, services *core.Services) {
	// Delete a Job
	// DELETE /jobs/{jobGuid}
	e.DELETE("/jobs/:guid", func(c echo.Context) error {
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		// look up the job
		job, err := services.Jobs.Get(guid)
		if err != nil {
			return c.JSON(
				http.StatusNotFound,
				"",
			)
		}

		// delete things associated with the job
		for _, schedule := range services.Schedules.ByJob(job) {
			err = workflows.DeletingASchedule(services, schedule, job)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, "")
			}
		}

		// actually delete the job
		err = services.Jobs.Delete(job)
		if err != nil {
			return c.JSON(
				http.StatusInternalServerError,
				"",
			)
		}

		return c.JSON(
			http.StatusNoContent,
			"",
		)
	})
}
