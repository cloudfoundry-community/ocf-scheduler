package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func AllCallSchedules(e *echo.Echo, services *core.Services) {
	// Get all schedules for a Call
	// GET /calls/{callGuid}/schedules
	e.GET("/calls/:guid/schedules", func(c echo.Context) error {
		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithGUID(c.Param("guid"))

		result := workflows.
			GettingAllCallSchedules.
			Call(input)

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case failures.AuthFailure:
				return c.JSON(http.StatusUnauthorized, "")
			default:
				return c.JSON(http.StatusNotFound, "")
			}
		}

		schedules := core.Inputify(result.Value()).ScheduleCollection

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
