package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

func DeleteCall(e *echo.Echo, services *core.Services) {
	// Delete a Call
	// DELETE /calls/{callGuid}
	e.DELETE("/calls/:guid", func(c echo.Context) error {
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

		// delete things associated with the call
		for _, schedule := range services.Schedules.ByCall(call) {
			err = workflows.DeletingASchedule(services, schedule, call)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, "")
			}
		}

		err = services.Calls.Delete(call)
		if err != nil {
			return c.JSON(
				http.StatusInternalServerError,
				"",
			)
		}

		return c.JSON(
			http.StatusNoContent,
			"",
		)
	})
}
