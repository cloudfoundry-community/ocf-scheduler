package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
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
		tag := "all-jobs"
		services.Logger.Info(tag, "trying to get all jobs")

		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			services.Logger.Error(tag, "authentication to this endpoint failed")
			return c.JSON(http.StatusUnauthorized, "")
		}

		spaceGUID := c.QueryParam("space_guid")

		jobs := services.Jobs.InSpace(spaceGUID)

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
