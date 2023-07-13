package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
	"github.com/cloudfoundry-community/ocf-scheduler/http/helpers"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

func CreateJobSchedule(e *echo.Echo, services *core.Services) {
	// Schedule a Job to run later
	// POST /jobs/{jobGuid}/schedules
	e.POST("/jobs/:guid/schedules", func(c echo.Context) error {
		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithSchedule(helpers.Schedulify(c)).
			WithGUID(c.Param("guid"))

		result := workflows.
			SchedulingAJob.
			Call(input)

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case failures.AuthFailure:
				return c.JSON(http.StatusUnauthorized, "")
			case failures.NoSuchJob:
				return c.JSON(http.StatusNotFound, "")
			default:
				return c.JSON(http.StatusUnprocessableEntity, "")
			}
		}

		schedule := core.Inputify(result.Value()).Schedule

		return c.JSON(
			http.StatusCreated,
			presenters.AsJobSchedule(schedule),
		)
	})
}
