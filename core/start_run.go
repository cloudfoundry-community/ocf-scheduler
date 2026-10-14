package core

import (
	"fmt"
	"time"
)

// StartRun marks execution started before a run service does any work. It
// tries up to attempts times, waiting backoff before the second attempt and
// doubling the wait after each failure, so a brief database error does not
// cost the run.
//
// When every attempt fails, StartRun returns false and the run is skipped. A
// missed run is better than a task or call whose outcome nothing records.
// Before it returns, StartRun records why on the execution and fails it with
// the same retries, so the row does not stay PENDING once the database
// answers again. If the database stays down throughout, the row can stay
// PENDING, and both errors are logged.
func StartRun(services *Services, execution *Execution, tag string, attempts int, backoff time.Duration) bool {
	err := retry(services, tag, attempts, backoff, fmt.Sprintf("mark execution %s started", execution.GUID), func() error {
		_, err := services.Executions.Start(execution)
		return err
	})
	if err == nil {
		return true
	}

	services.Logger.Error(tag, fmt.Sprintf("skipping execution %s, because it could not be marked started after %d attempts: %v", execution.GUID, attempts, err))

	// The message is set on the execution before it is written, and Fail
	// writes the whole row, so a Fail that lands stores the message too.
	if _, err := services.Executions.UpdateMessage(execution, "the run was skipped, because its execution could not be marked started"); err != nil {
		services.Logger.Error(tag, fmt.Sprintf("could not record why execution %s was skipped: %v", execution.GUID, err))
	}

	err = retry(services, tag, attempts, backoff, fmt.Sprintf("fail skipped execution %s", execution.GUID), func() error {
		_, err := services.Executions.Fail(execution)
		return err
	})
	if err != nil {
		services.Logger.Error(tag, fmt.Sprintf("could not fail skipped execution %s after %d attempts, so it stays PENDING: %v", execution.GUID, attempts, err))
	}

	return false
}

// retry calls try up to attempts times with a doubling backoff, logging each
// failure but the last, and returns the last error.
func retry(services *Services, tag string, attempts int, backoff time.Duration, what string, try func() error) error {
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err = try(); err == nil {
			return nil
		}

		if attempt == attempts {
			break
		}

		services.Logger.Warn(tag, fmt.Sprintf("cannot %s (attempt %d of %d), retrying in %s: %v", what, attempt, attempts, backoff, err))
		time.Sleep(backoff)
		backoff *= 2
	}

	return err
}
