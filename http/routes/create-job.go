package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/workflows"
)

func CreateJob(e *echo.Echo, services *core.Services) {
	// Create Job
	// POST /jobs?app_guid=string
	e.POST("/jobs", func(c echo.Context) error {
		result := workflows.
			CreatingAJob.
			Call(core.NewInput(c, services))

		if result.Failure() {
			switch core.Causify(result.Error()) {
			case "auth-failure":
				return c.JSON(http.StatusUnauthorized, "")
			default:
				return c.JSON(http.StatusUnprocessableEntity, "")
			}
		}

		job, _ := core.Inputify(result.Value()).Executable.ToJob()

		return c.JSON(
			http.StatusCreated,
			job,
		)
	})
}
