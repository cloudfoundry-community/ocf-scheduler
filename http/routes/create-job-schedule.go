package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func CreateJobSchedule(e *echo.Echo, services *core.Services) {
	// Schedule a Job to run later
	// POST /jobs/{jobGuid}/schedules
	e.POST("/jobs/:guid/schedules", func(c echo.Context) error {
		result := workflows.
			SchedulingAJob.
			Call(core.NewInput(c, services))

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case "auth-failure":
				return c.JSON(http.StatusUnauthorized, "")
			case "no-such-job":
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
