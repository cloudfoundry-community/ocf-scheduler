package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
	"github.com/cloudfoundry-community/ocf-scheduler/http/helpers"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

func CreateJob(e *echo.Echo, services *core.Services) {
	// Create Job
	// POST /jobs?app_guid=string
	e.POST("/jobs", func(c echo.Context) error {
		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithExecutable(helpers.Jobify(c)).
			WithAppGUID(c.QueryParam("app_guid"))

		result := workflows.
			CreatingAJob.
			Call(input)

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case failures.AuthFailure:
				return c.JSON(http.StatusUnauthorized, "")
			default:
				return c.JSON(http.StatusUnprocessableEntity, "")
			}
		}

		job, _ := core.Inputify(result.Value()).Executable.ToJob()

		return c.JSON(
			http.StatusCreated,
			job,
		)
	})
}
