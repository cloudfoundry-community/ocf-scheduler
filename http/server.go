package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/routes"
)

// Server serves the scheduler API on bind. When broker is not nil, it also
// serves the Open Service Broker API under /v2, which checks its own basic
// auth credentials instead of a Cloud Foundry token.
func Server(bind string, services *core.Services, broker http.Handler) *http.Server {
	e := echo.New()

	// not technically necessary, just part of my default API skeleton
	e.GET("/", func(c echo.Context) error {
		return c.JSON(
			http.StatusOK,
			[]byte("{}"),
		)
	})

	routes.Apply(e, services)

	if broker != nil {
		e.Any("/v2/*", echo.WrapHandler(broker))
	}

	server := e.Server
	server.Addr = bind

	return server
}
