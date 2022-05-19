package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func DeleteJob(e *echo.Echo, services *core.Services) {
	// Delete a Job
	// DELETE /jobs/{jobGuid}
	e.DELETE("/jobs/:guid", func(c echo.Context) error {
		input := core.NewInput(services).
			WithAuth(c.Request().Header.Get(echo.HeaderAuthorization)).
			WithGUID(c.Param("guid"))

		result := workflows.
			DeletingAJob.
			Call(input)

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case failures.AuthFailure:
				return c.JSON(http.StatusUnauthorized, "")
			case failures.NoSuchJob:
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
