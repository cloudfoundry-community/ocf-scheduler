package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
)

func AllCallExecutions(e *echo.Echo, services *core.Services) {
	// Get all execution histories for a Call
	// GET /calls/{callGuid}/history
	e.GET("/calls/:guid/history", func(c echo.Context) error {
		tag := "all-call-executions"

		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		guid := c.Param("guid")

		call, err := services.Calls.Get(guid)
		if err != nil {
			return c.JSON(missing(services, tag, auth, http.StatusNotFound), "")
		}

		if status := authorize(services, tag, auth, call.SpaceGUID, http.StatusNotFound); status != 0 {
			return c.JSON(status, "")
		}

		executions := services.Executions.ByCall(call)

		services.Logger.Debug(tag, fmt.Sprintf("got %d executions", len(executions)))

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
