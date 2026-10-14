package core

import (
	"fmt"
	"runtime/debug"
)

// RecoverRun is deferred at the top of the work a RunService submits to the
// worker pool. The pool runs that work on its own goroutines and does not
// recover panics, so a panic in one run would end the scheduler. RecoverRun
// logs the panic with its stack trace, fails the execution, and lets the
// worker carry on with the next run. JobRun and CallRun skip a run whose
// execution could not be stored, but a nil execution is still guarded, so a
// future caller cannot turn a recovered panic into a crash.
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

	if _, err := services.Executions.UpdateMessage(execution, fmt.Sprintf("run failed unexpectedly: %v", recovered)); err != nil {
		services.Logger.Error(tag, fmt.Sprintf("could not record the panic on execution %s: %v", execution.GUID, err))
	}
	if _, err := services.Executions.Fail(execution); err != nil {
		services.Logger.Error(tag, fmt.Sprintf("could not fail execution %s after a panic, so it is left unfinished: %v", execution.GUID, err))
	}
}
