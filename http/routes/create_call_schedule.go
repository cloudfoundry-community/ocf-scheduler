package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func CreateCallSchedule(e *echo.Echo, services *core.Services) {
	// Schedule a Call to run later
	// POST /calls/{callGuid}/schedules
	e.POST("/calls/:guid/schedules", func(c echo.Context) error {
		result := workflows.
			SchedulingACall.
			Call(core.NewInput(c, services))

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case "auth-failure":
				return c.JSON(http.StatusUnauthorized, "")
			case "no-such-call":
				return c.JSON(http.StatusNotFound, "")
			default:
				return c.JSON(http.StatusUnprocessableEntity, "")
			}
		}

		schedule := core.Inputify(result.Value()).Schedule

		return c.JSON(
			http.StatusCreated,
			presenters.AsCallSchedule(schedule),
		)
	})
}
