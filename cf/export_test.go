package cf

import "time"

// SetTaskPollInterval shortens waitForTask's sleep for a test and returns a
// function that restores it.
func SetTaskPollInterval(interval time.Duration) func() {
	saved := taskPollInterval
	taskPollInterval = interval
	return func() { taskPollInterval = saved }
}

// SetTaskGUIDRetries changes how Execute retries storing a task GUID for a
// test and returns a function that restores the defaults.
func SetTaskGUIDRetries(attempts int, backoff time.Duration) func() {
	savedAttempts, savedBackoff := taskGUIDAttempts, taskGUIDBackoff
	taskGUIDAttempts, taskGUIDBackoff = attempts, backoff
	return func() { taskGUIDAttempts, taskGUIDBackoff = savedAttempts, savedBackoff }
}

// SetTaskPendingLimit changes how long waitForTask follows a PENDING task
// for a test and returns a function that restores it.
func SetTaskPendingLimit(limit time.Duration) func() {
	saved := taskPendingLimit
	taskPendingLimit = limit
	return func() { taskPendingLimit = saved }
}
