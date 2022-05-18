package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func AllCallExecutions(e *echo.Echo, services *core.Services) {
	// Get all execution histories for a Call
	// GET /calls/{callGuid}/history
	e.GET("/calls/:guid/history", func(c echo.Context) error {
		result := workflows.
			GettingCallExecutions.
			Call(core.NewInput(c, services))

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case "auth-failure":
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

type callExecutionCollection struct {
	Pagination *pagination                 `json:"pagination"`
	Resources  []*presenters.CallExecution `json:"resources"`
}
