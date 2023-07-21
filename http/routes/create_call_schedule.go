package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
)

func CreateCallSchedule(e *echo.Echo, services *core.Services) {
	// Schedule a Call to run later
	// POST /calls/{callGuid}/schedules
	e.POST("/calls/:guid/schedules", func(c echo.Context) error {
		tag := "create-call-schedule"

		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
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

		if services.Cron.Validate(input.Expression) != nil {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

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
