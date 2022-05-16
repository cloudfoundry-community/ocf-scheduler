package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func DeleteCall(e *echo.Echo, services *core.Services) {
	// Delete a Call
	// DELETE /calls/{callGuid}
	e.DELETE("/calls/:guid", func(c echo.Context) error {
		result := workflows.
			DeletingACall.
			Call(core.NewInput(c, services))

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case "auth-failure":
				return c.JSON(http.StatusUnauthorized, "")
			case "no-such-call":
				return c.JSON(http.StatusNotFound, "")
			default:
				return c.JSON(http.StatusInternalServerError, "")
			}
		}

		return c.JSON(
			http.StatusNoContent,
			"",
		)
	})
}
