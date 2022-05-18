package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func DeleteCallSchedule(e *echo.Echo, services *core.Services) {
	// Delete the given schedule for the given Call
	// DELETE /calls/{callGuid}/schedules/{scheduleGuid}
	e.DELETE("/calls/:guid/schedules/:schedule_guid", func(c echo.Context) error {
		result := workflows.
			UnschedulingACall.
			Call(core.NewInput(c, services))

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case "auth-failure":
				return c.JSON(http.StatusUnauthorized, "")
			case "no-such-call", "no-such-schedule":
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
