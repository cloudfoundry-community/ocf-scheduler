package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
)

func AllJobExecutions(e *echo.Echo, services *core.Services) {
	// Get all execution histories for a Job
	// GET /jobs/{jobGuid}/history
	e.GET("/jobs/:guid/history", func(c echo.Context) error {
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		job, err := services.Jobs.Get(guid)
		if err != nil {
			return c.JSON(http.StatusNotFound, "")
		}

		executions := services.Executions.ByJob(job)

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

type jobExecutionCollection struct {
	Pagination *pagination                `json:"pagination"`
	Resources  []*presenters.JobExecution `json:"resources"`
}
