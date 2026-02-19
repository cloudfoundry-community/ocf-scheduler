package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/presenters"
)

func ExecuteCall(e *echo.Echo, services *core.Services) {
	// Execute a Call as soon as possible
	// POST /calls/{callGuid}/execute
	e.POST("/calls/:guid/execute", func(c echo.Context) error {
		tag := "execute-call"
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

		input := &core.Execution{}

		if err = c.Bind(&input); err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to parse execution request for call %s: %v", guid, err))
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		input.RefGUID = guid
		input.RefType = "call"

		execution, err := services.Executions.Persist(input)
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to persist execution for call %s: %v", guid, err))
			return c.JSON(http.StatusUnprocessableEntity, "")
		}

		services.Runner.Execute(services, execution, call)
		services.Logger.Info(tag, fmt.Sprintf("dispatched execution for call %s", guid))

		return c.JSON(
			http.StatusCreated,
			presenters.AsCallExecution(execution),
		)
	})
}
