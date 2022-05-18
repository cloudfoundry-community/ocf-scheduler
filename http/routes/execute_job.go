package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func ExecuteJob(e *echo.Echo, services *core.Services) {
	// Execute a Job as soon as possible
	// POST /jobs/{jobGuid}/execute
	e.POST("/jobs/:guid/execute", func(c echo.Context) error {
		result := workflows.
			ExecutingAJob.
			Call(core.NewInput(c, services))

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case "auth-failure":
				return c.JSON(http.StatusUnauthorized, "")
			case "no-such-job":
				return c.JSON(http.StatusNotFound, "")
			default:
				return c.JSON(http.StatusUnprocessableEntity, "")
			}
		}

		execution := core.Inputify(result.Value()).Execution

		return c.JSON(
			http.StatusCreated,
			presenters.AsJobExecution(execution),
		)
	})
}
