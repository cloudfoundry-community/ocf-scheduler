package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

func DeleteCall(e *echo.Echo, services *core.Services) {
	// Delete a Call
	// DELETE /calls/{callGuid}
	e.DELETE("/calls/:guid", func(c echo.Context) error {
		tag := "delete-call"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			services.Logger.Error(tag, "authentication failed")
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		call, err := services.Calls.Get(guid)
		if err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("call %s not found", guid))
			return c.JSON(
				http.StatusNotFound,
				"",
			)
		}

		// delete things associated with the call
		for _, schedule := range services.Schedules.ByCall(call) {
			err = workflows.DeletingASchedule(services, schedule, call)
			if err != nil {
				services.Logger.Error(tag, fmt.Sprintf("failed to delete schedule %s for call %s: %v", schedule.GUID, guid, err))
				return c.JSON(http.StatusInternalServerError, "")
			}
		}

		err = services.Calls.Delete(call)
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to delete call %s: %v", guid, err))
			return c.JSON(
				http.StatusInternalServerError,
				"",
			)
		}

		services.Logger.Info(tag, fmt.Sprintf("deleted call %s", guid))
		return c.JSON(
			http.StatusNoContent,
			"",
		)
	})
}
