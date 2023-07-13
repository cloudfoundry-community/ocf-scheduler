package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/core/failures"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

func AllCallScheduleExecutions(e *echo.Echo, services *core.Services) {
	// Get all execution histories for a Call and Schedule
	// GET /calls/{callGuid}/schedules/{scheduleGuid}/history
	e.GET("/calls/:guid/schedules/:schedule_guid/history", func(c echo.Context) error {
		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithGUID(c.Param("guid")).
			WithScheduleGUID(c.Param("guid"))

		result := workflows.
			GettingScheduledCallExecutions.
			Call(input)

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case failures.AuthFailure:
				return c.JSON(http.StatusUnauthorized, "")
			default:
				return c.JSON(http.StatusNotFound, "")
			}
		}

		executions := core.Inputify(result.Value()).ExecutionCollection

		output := &callExecutionCollection{
			Resources: presenters.AsCallExecutionCollection(executions),
			Pagination: &pagination{
				TotalPages:   1,
				TotalResults: len(executions),
				First:        &pageref{Href: "first"},
				Last:         &pageref{Href: "last"},
				Next:         &pageref{Href: "next"},
				Previous:     &pageref{Href: "previous"},
			},
		}

		return c.JSON(
			http.StatusOK,
			output,
		)
	})
}
