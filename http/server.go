package http

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func verifyAuth(c echo.Context) error {
	auth := c.Request().Header.Get(echo.HeaderAuthorization)

	return fmt.Errorf("unimplemented")
}

func Server(bind string, services *core.Services) *http.Server {
	e := echo.New()

	// not technically necessary, just part of my default API skeleton
	e.GET("/health", func(c echo.Context) error {
		return c.String(
			http.StatusOK,
			"OK",
		)
	})

	// CALL ROUTES

	// JOB ROUTES

	server := e.Server
	server.Addr = bind

	return server
}
