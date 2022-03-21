package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/auth"
)

func DeleteJob(e *echo.Echo, services *core.Services) {
	// Delete a Job
	// DELETE /jobs/{jobGuid}
	e.DELETE("/jobs/:guid", func(c echo.Context) error {
		if auth.Verify(c) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		job, err := services.Jobs.Get(guid)
		if err != nil {
			return c.JSON(
				http.StatusNotFound,
				"",
			)
		}

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
