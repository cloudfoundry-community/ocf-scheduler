package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
)

func AllJobScheduleExecutions(e *echo.Echo, services *core.Services) {
	// Get all execution histories for a Job and Schedule
	// GET /jobs/{jobGuid}/schedules/{scheduleGuid}/history
	e.GET("/jobs/:guid/schedules/:schedule_guid/history", func(c echo.Context) error {
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		_, err := services.Jobs.Get(guid)
		if err != nil {
			return c.JSON(http.StatusNotFound, "")
		}

		scheduleGUID := c.Param("schedule_guid")
		schedule, err := services.Schedules.Get(scheduleGUID)
		if err != nil {
			return c.JSON(http.StatusNotFound, "")
		}

		executions := services.Executions.BySchedule(schedule)

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
