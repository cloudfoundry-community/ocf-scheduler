package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

type pageref struct {
	Href string `json:"href"`
}

type pagination struct {
	First        *pageref `json:"first"`
	Last         *pageref `json:"last"`
	Next         *pageref `json:"next"`
	Previous     *pageref `json:"previous"`
	TotalPages   int      `json:"total_pages"`
	TotalResults int      `json:"total_results"`
}

type jobCollection struct {
	Pagination *pagination `json:"pagination"`
	Resources  []*core.Job `json:"resources"`
}

func AllJobs(e *echo.Echo, services *core.Services) {
	// Get all Jobs within space
	// GET /jobs?space_guid=string
	e.GET("/jobs", func(c echo.Context) error {
		result := workflows.
			GettingAllJobs.
			Call(core.NewInput(c, services))

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
