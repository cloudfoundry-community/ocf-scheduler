package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func AllJobs(e *echo.Echo, services *core.Services) {
	// Get all Jobs within space
	// GET /jobs?space_guid=string
	e.GET("/jobs", func(c echo.Context) error {
		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithSpaceGUID(c.QueryParam("space_guid"))

		result := workflows.
			GettingAllJobs.
			Call(input)

		if result.Failure() {
			return c.JSON(http.StatusUnauthorized, "")
		}

		jobs := core.Inputify(result.Value()).JobCollection

		output := &jobCollection{
			Resources: jobs,
			Pagination: &pagination{
				TotalPages:   1,
				TotalResults: len(jobs),
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

type jobCollection struct {
	Pagination *pagination `json:"pagination"`
	Resources  []*core.Job `json:"resources"`
}
