package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
)

func AllJobScheduleExecutions(e *echo.Echo, services *core.Services) {
	// Get all execution histories for a Job and Schedule
	// GET /jobs/{jobGuid}/schedules/{scheduleGuid}/history
	e.GET("/jobs/:guid/schedules/:schedule_guid/history", func(c echo.Context) error {
		tag := "all-job-schedule-executions"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		guid := c.Param("guid")

		job, err := services.Jobs.Get(guid)
		if err != nil {
			return c.JSON(missing(services, tag, auth, http.StatusNotFound), "")
		}

		if status := authorize(services, tag, auth, job.SpaceGUID, http.StatusNotFound); status != 0 {
			return c.JSON(status, "")
		}

		scheduleGUID := c.Param("schedule_guid")
		schedule, err := services.Schedules.Get(scheduleGUID)
		if err != nil {
			return c.JSON(http.StatusNotFound, "")
		}

		if !ownedBy(schedule, "job", guid) {
			services.Logger.Warn(tag, "schedule "+scheduleGUID+" does not belong to job "+guid)
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
