package routes

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http/auth"
)

func ExecuteCall(e *echo.Echo, services *core.Services) {
	// Execute a Call as soon as possible
	// POST /calls/{callGuid}/execute
	e.POST("/calls/:guid/execute", func(c echo.Context) error {
		if auth.Verify(c) != nil {
			return c.JSON(http.StatusUnauthorized, "")
		}

		now := time.Now().UTC()

		input := struct {
			Text string `json:"text"`
		}{}

		if err := c.Bind(&input); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, &input)
		}

		return c.JSON(
			http.StatusOK,
			"POST RESULT",
		)
	})
}
