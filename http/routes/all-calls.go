package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

type callCollection struct {
	Pagination *pagination  `json:"pagination"`
	Resources  []*core.Call `json:"resources"`
}

func AllCalls(e *echo.Echo, services *core.Services) {
	// Get all Calls within space
	// GET /calls?space_guid=string
	e.GET("/calls", func(c echo.Context) error {
		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithSpaceGUID(c.QueryParam("space_guid"))

		result := workflows.
			GettingAllCalls.
			Call(input)

		if result.Failure() {
			return c.JSON(http.StatusUnauthorized, "")
		}

		calls := core.Inputify(result.Value()).CallCollection

		output := &callCollection{
			Resources: calls,
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
