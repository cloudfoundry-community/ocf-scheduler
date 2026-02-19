package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
)

func ExecuteJob(e *echo.Echo, services *core.Services) {
	// Execute a Job as soon as possible
	// POST /jobs/{jobGuid}/execute
	e.POST("/jobs/:guid/execute", func(c echo.Context) error {
		tag := "execute-job"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			services.Logger.Error(tag, "authentication failed")
			return c.JSON(http.StatusUnauthorized, "")
		}

		guid := c.Param("guid")

		job, err := services.Jobs.Get(guid)
		if err != nil {
			services.Logger.Warn(tag, fmt.Sprintf("job %s not found", guid))
			return c.JSON(
				http.StatusNotFound,
				"",
			)
		}

		input := &core.Execution{}

		if err = c.Bind(&input); err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to parse execution request for job %s: %v", guid, err))
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		input.RefGUID = guid
		input.RefType = "job"

		execution, err := services.Executions.Persist(input)
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to persist execution for job %s: %v", guid, err))
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		services.Runner.Execute(services, execution, job)
		services.Logger.Info(tag, fmt.Sprintf("dispatched execution for job %s", guid))

		return c.JSON(
			http.StatusCreated,
			presenters.AsJobExecution(execution),
		)
	})
}
