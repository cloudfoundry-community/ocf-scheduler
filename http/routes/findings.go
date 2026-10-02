package routes

import (
	"fmt"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// requestError is a 4xx body for a problem with the request itself.
func requestError(code, format string, args ...any) core.Findings {
	return core.Findings{
		Errors:   []core.Finding{{Code: code, Message: fmt.Sprintf(format, args...)}},
		Warnings: []core.Finding{},
	}
}
