package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/auth"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
)

func AllJobSchedules(e *echo.Echo, services *core.Services) {
	// Get all schedules for a Job
	// GET /jobs/{jobGuid}/schedules
	e.GET("/jobs/:guid/schedules", func(c echo.Context) error {
		if auth.Verify(c) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		job, err := services.Jobs.Get(guid)
		if err != nil {
			return c.JSON(http.StatusNotFound, "")
		}

		schedules := services.Schedules.ByJob(job)

		output := &jobScheduleCollection{
			Resources: presenters.AsJobScheduleCollection(schedules),
			Pagination: &pagination{
				TotalPages:   1,
				TotalResults: len(schedules),
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

type jobScheduleCollection struct {
	Pagination *pagination               `json:"pagination"`
	Resources  []*presenters.JobSchedule `json:"resources"`
}
