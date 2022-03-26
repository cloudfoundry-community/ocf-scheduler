package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/auth"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
)

func CreateCallSchedule(e *echo.Echo, services *core.Services) {
	// Schedule a Call to run later
	// POST /calls/{callGuid}/schedules
	e.POST("/calls/:guid/schedules", func(c echo.Context) error {
		tag := "create-call-schedule"

		if auth.Verify(c) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		call, err := services.Calls.Get(guid)
		if err != nil {
			return c.JSON(
				http.StatusNotFound,
				"",
			)
		}

		input := &core.Schedule{}

		if err = c.Bind(&input); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		input.RefGUID = guid
		input.RefType = "call"

		services.Logger.Info(tag, fmt.Sprintf("expression == '%s', expression_type == '%s'", input.Expression, input.ExpressionType))

		schedule, err := services.Schedules.Persist(input)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		services.Cron.Add(core.NewCallRun(call, schedule, services))

		return c.JSON(
			http.StatusCreated,
			presenters.AsCallSchedule(schedule),
		)
	})
}
