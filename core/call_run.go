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

	// A run with no stored execution has nothing to record its outcome on,
	// so it is skipped rather than run unrecorded.
	execution, err := run.services.Executions.Persist(execution)
	if err != nil {
		run.services.Logger.Error("call-run", fmt.Sprintf("skipping a run of call %s on schedule %s, because its execution could not be stored: %v", run.call.GUID, run.schedule.GUID, err))
		return
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
