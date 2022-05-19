package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func AllJobScheduleExecutions(e *echo.Echo, services *core.Services) {
	// Get all execution histories for a Job and Schedule
	// GET /jobs/{jobGuid}/schedules/{scheduleGuid}/history
	e.GET("/jobs/:guid/schedules/:schedule_guid/history", func(c echo.Context) error {
		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithGUID(c.Param("guid")).
			WithScheduleGUID(c.Param("schedule_guid"))

		result := workflows.
			GettingScheduledJobExecutions.
			Call(input)

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case "auth-failure":
				return c.JSON(http.StatusUnauthorized, "")
			default:
				return c.JSON(http.StatusNotFound, "")
			}
		}

		executions := core.Inputify(result.Value()).ExecutionCollection

		output := &jobExecutionCollection{
			Resources: presenters.AsJobExecutionCollection(executions),
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
