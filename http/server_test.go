package http

import (
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// A client that sends its request slowly, or stops sending, must not hold a
// connection open forever.
func TestServerTimesOutSlowClients(t *testing.T) {
	server := Server("127.0.0.1:0", &core.Services{}, nil)

	if server.ReadHeaderTimeout <= 0 || server.ReadTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Errorf("want read header, read, and idle timeouts; got %s, %s, and %s",
			server.ReadHeaderTimeout, server.ReadTimeout, server.IdleTimeout)
	}
}
