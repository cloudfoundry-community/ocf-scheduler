package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
)

func AllCallSchedules(e *echo.Echo, services *core.Services) {
	// Get all schedules for a Call
	// GET /calls/{callGuid}/schedules
	e.GET("/calls/:guid/schedules", func(c echo.Context) error {
		tag := "all-call-schedules"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		guid := c.Param("guid")

		call, err := services.Calls.Get(guid)
		if err != nil {
			return c.JSON(missing(services, tag, auth, http.StatusNotFound), "")
		}

		if status := authorize(services, tag, auth, call.SpaceGUID, http.StatusNotFound); status != 0 {
			return c.JSON(status, "")
		}

		schedules, err := services.Schedules.ByCall(call)
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("error retrieving schedules for call %s: %v", guid, err))
			return c.JSON(http.StatusInternalServerError, "error retrieving schedules")
		}

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
