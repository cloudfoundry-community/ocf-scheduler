package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func DeleteJobSchedule(e *echo.Echo, services *core.Services) {
	// Delete the given schedule for the given Job
	// DELETE /jobs/{jobGuid}/schedules/{scheduleGuid}
	e.DELETE("/jobs/:guid/schedules/:schedule_guid", func(c echo.Context) error {
		result := workflows.
			UnschedulingAJob.
			Call(core.NewInput(c, services))

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case "auth-failure":
				return c.JSON(http.StatusUnauthorized, "")
			case "no-such-job", "no-such-schedule":
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
