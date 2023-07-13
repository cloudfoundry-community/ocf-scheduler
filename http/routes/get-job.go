package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

func GetJob(e *echo.Echo, services *core.Services) {
	// Get a Job (sha-na-na-na, sha-na-na-na-na, ahh-do)
	// GET /jobs/{jobGuid}
	e.GET("/jobs/:guid", func(c echo.Context) error {
		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithGUID(c.Param("guid"))

		result := workflows.
			GettingAJob.
			Call(input)

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case failures.AuthFailure:
				return c.JSON(http.StatusUnauthorized, "")
			default:
				return c.JSON(http.StatusNotFound, "")
			}
		}

		job, _ := core.Inputify(result.Value()).Executable.ToJob()

		return c.JSON(
			http.StatusOK,
			job,
		)
	})
}
