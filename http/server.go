package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/routes"
)

func Server(bind string, services *core.Services) *http.Server {
	e := echo.New()

	// not technically necessary, just part of my default API skeleton
	e.GET("/", func(c echo.Context) error {
		return c.JSON(
			http.StatusOK,
			"{}",
		)
	})

	routes.Apply(e, services)

	server := e.Server
	server.Addr = bind

	return server
}
