package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
)

type callCollection struct {
	Pagination *pagination        `json:"pagination"`
	Resources  []*presenters.Call `json:"resources"`
}

func AllCalls(e *echo.Echo, services *core.Services) {
	// Get all Calls within space
	// GET /calls?space_guid=string
	e.GET("/calls", func(c echo.Context) error {
		tag := "all-calls"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		spaceGUID := c.QueryParam("space_guid")

		if status := authorize(services, tag, auth, spaceGUID, http.StatusForbidden); status != 0 {
			return c.JSON(status, "")
		}

		calls := services.Calls.InSpace(spaceGUID)

		output := &callCollection{
			Resources: presenters.AsCallCollection(calls),
			Pagination: &pagination{
				TotalPages:   1,
				TotalResults: len(calls),
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
