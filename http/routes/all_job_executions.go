package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func AllJobExecutions(e *echo.Echo, services *core.Services) {
	// Get all execution histories for a Job
	// GET /jobs/{jobGuid}/history
	e.GET("/jobs/:guid/history", func(c echo.Context) error {
		result := workflows.
			GettingScheduledJobExecutions.
			Call(core.NewInput(c, services))

		if result.Failure() {
			cause := result.Error().(string)

			switch cause {
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

type jobExecutionCollection struct {
	Pagination *pagination                `json:"pagination"`
	Resources  []*presenters.JobExecution `json:"resources"`
}
