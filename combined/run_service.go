package combined

import (
	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// A combined RunService implementation that dispatches to a more specific
// RunService implementation based on context.
type RunService struct {
	dispatch map[string]core.RunService
}

func NewRunService(dispatch map[string]core.RunService) *RunService {
	return &RunService{
		dispatch: dispatch,
	}
}

func (service *RunService) Execute(services *core.Services, execution *core.Execution, executable core.Executable) {
	service.dispatch[executable.Type()].Execute(services, execution, executable)
}
