package core

import (
	"fmt"
	"runtime/debug"
)

// RecoverRun is deferred at the top of the work a RunService submits to the
// worker pool. The pool runs that work on its own goroutines and does not
// recover panics, so a panic in one run would end the scheduler. RecoverRun
// logs the panic with its stack trace, fails the execution when there is
// one, and lets the worker carry on with the next run.
func RecoverRun(services *Services, execution *Execution, tag string) {
	recovered := recover()
	if recovered == nil {
		return
	}

	if execution == nil {
		services.Logger.Error(tag, fmt.Sprintf("run panicked with no execution: %v\n%s", recovered, debug.Stack()))
		return
	}

	services.Logger.Error(tag, fmt.Sprintf("run panicked for execution %s: %v\n%s", execution.GUID, recovered, debug.Stack()))

	// Failing the execution talks to the database, and a second panic here
	// must not escape the worker either.
	defer func() {
		if again := recover(); again != nil {
			services.Logger.Error(tag, fmt.Sprintf("could not fail execution %s after a panic: %v", execution.GUID, again))
		}
	}()

	services.Executions.UpdateMessage(execution, fmt.Sprintf("run failed unexpectedly: %v", recovered))
	services.Executions.Fail(execution)
}
