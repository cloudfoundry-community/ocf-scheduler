package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/auth"
	"github.com/starkandwayne/scheduler-for-ocf/http/presenters"
)

func ExecuteJob(e *echo.Echo, services *core.Services) {
	// Execute a Job as soon as possible
	// POST /jobs/{jobGuid}/execute
	e.POST("/jobs/:guid/execute", func(c echo.Context) error {
		if auth.Verify(c) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		job, err := services.Jobs.Get(guid)
		if err != nil {
			return c.JSON(
				http.StatusNotFound,
				"",
			)
		}

		input := &core.Execution{}

		if err = c.Bind(&input); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		input.RefGUID = guid
		input.RefType = "job"

		execution, err := services.Executions.Persist(input)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		services.Runner.Execute(services, execution, job)

		return c.JSON(
			http.StatusCreated,
			presenters.AsJobExecution(execution),
		)
	})
}
