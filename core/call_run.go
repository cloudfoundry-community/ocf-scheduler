package core

import "fmt"

// CallRun is a struct that wraps a Call and makes it Runnable.
type CallRun struct {
	call     *Call
	schedule *Schedule
	services *Services
}

func NewCallRun(call *Call, schedule *Schedule, services *Services) *CallRun {
	return &CallRun{
		call,
		schedule,
		services,
	}
}

func (run *CallRun) Run() {
	execution := &Execution{
		RefGUID:      run.call.GUID,
		RefType:      "call",
		ScheduleGUID: run.schedule.GUID,
	}

	execution, err := run.services.Executions.Persist(execution)
	if err != nil {
		run.services.Logger.Warn("call-run", fmt.Sprintf("failed to persist execution for call %s: %v", run.call.GUID, err))
	}

	run.services.Runner.Execute(
		run.services,
		execution,
		run.call,
	)
}

func (run *CallRun) Services() *Services {
	return run.services
}

func (run *CallRun) Job() *Job {
	return nil
}

func (run *CallRun) Call() *Call {
	return run.call
}

func (run *CallRun) Schedule() *Schedule {
	return run.schedule
}
