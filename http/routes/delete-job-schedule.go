package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

func DeleteJobSchedule(e *echo.Echo, services *core.Services) {
	// Delete the given schedule for the given Job
	// DELETE /jobs/{jobGuid}/schedules/{scheduleGuid}
	e.DELETE("/jobs/:guid/schedules/:schedule_guid", func(c echo.Context) error {
		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithGUID(c.Param("guid")).
			WithScheduleGUID(c.Param("schedule_guid"))

		result := workflows.
			UnschedulingAJob.
			Call(input)

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case failures.AuthFailure:
				return c.JSON(http.StatusUnauthorized, "")
			case failures.NoSuchJob, failures.NoSuchSchedule:
				return c.JSON(http.StatusNotFound, "")
			default:
				return c.JSON(http.StatusInternalServerError, "")
			}
		}

		return c.JSON(
			http.StatusNoContent,
			"",
		)
	})
}
