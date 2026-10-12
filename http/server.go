package http

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/http/routes"
)

// The server's timeouts stop a client that sends its request slowly, or
// stops sending, from holding a connection and its handler open. No route
// reads a body larger than a small JSON document.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	idleTimeout       = 2 * time.Minute
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
	server.ReadHeaderTimeout = readHeaderTimeout
	server.ReadTimeout = readTimeout
	server.IdleTimeout = idleTimeout

	return server
}
