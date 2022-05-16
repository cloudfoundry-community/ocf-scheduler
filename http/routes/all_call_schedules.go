package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
)

func AllCallSchedules(e *echo.Echo, services *core.Services) {
	// Get all schedules for a Call
	// GET /calls/{callGuid}/schedules
	e.GET("/calls/:guid/schedules", func(c echo.Context) error {
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		call, err := services.Calls.Get(guid)
		if err != nil {
			return c.JSON(http.StatusNotFound, "")
		}

		schedules := services.Schedules.ByRef("call", call.GUID)

		output := &callScheduleCollection{
			Resources: presenters.AsCallScheduleCollection(schedules),
			Pagination: &pagination{
				TotalPages:   1,
				TotalResults: len(schedules),
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

type callScheduleCollection struct {
	Pagination *pagination                `json:"pagination"`
	Resources  []*presenters.CallSchedule `json:"resources"`
}
