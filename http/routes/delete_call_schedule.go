package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

func DeleteCallSchedule(e *echo.Echo, services *core.Services) {
	// Delete the given schedule for the given Call
	// DELETE /calls/{callGuid}/schedules/{scheduleGuid}
	e.DELETE("/calls/:guid/schedules/:schedule_guid", func(c echo.Context) error {
		tag := "delete-call-schedule"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			services.Logger.Error(tag, "authentication failed")
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		call, err := services.Calls.Get(guid)
		if err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("call %s not found", guid))
			return c.JSON(http.StatusNotFound, "")
		}

		scheduleGUID := c.Param("schedule_guid")
		schedule, err := services.Schedules.Get(scheduleGUID)
		if err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("schedule %s not found for call %s", scheduleGUID, guid))
			return c.JSON(http.StatusNotFound, "")
		}

		err = workflows.DeletingASchedule(services, schedule, call)
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to delete schedule %s for call %s: %v", scheduleGUID, guid, err))
			return c.JSON(http.StatusInternalServerError, "")
		}

		services.Logger.Info(tag, fmt.Sprintf("deleted schedule %s for call %s", scheduleGUID, guid))
		return c.JSON(
			http.StatusNoContent,
			"",
		)
	})
}
