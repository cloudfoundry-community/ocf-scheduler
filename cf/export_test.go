package cf

import "time"

// SetTaskPollInterval shortens waitForTask's sleep for a test and returns a
// function that restores it.
func SetTaskPollInterval(interval time.Duration) func() {
	saved := taskPollInterval
	taskPollInterval = interval
	return func() { taskPollInterval = saved }
}
