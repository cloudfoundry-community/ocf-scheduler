package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func GetJob(e *echo.Echo, services *core.Services) {
	// Get a Job (sha-na-na-na, sha-na-na-na-na, ahh-do)
	// GET /jobs/{jobGuid}
	e.GET("/jobs/:guid", func(c echo.Context) error {
		tag := "get-job"
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

		return c.JSON(
			http.StatusOK,
			job,
		)
	})
}
