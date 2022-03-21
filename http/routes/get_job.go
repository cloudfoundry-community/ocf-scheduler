package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/auth"
)

func GetJob(e *echo.Echo, services *core.Services) {
	// Get a Job (sha-na-na-na, sha-na-na-na-na, ahh-do)
	// GET /jobs/{jobGuid}
	e.GET("/jobs/:guid", func(c echo.Context) error {
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

		return c.JSON(
			http.StatusOK,
			job,
		)
	})
}
