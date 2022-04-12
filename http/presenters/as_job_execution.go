package presenters

import (
	"time"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func AsJobExecution(execution *core.Execution) *JobExecution {
	output := &JobExecution{
		GUID:          execution.GUID,
		JobGUID:       execution.RefGUID,
		TaskGUID:      execution.TaskGUID,
		ScheduleGUID:  execution.ScheduleGUID,
		ScheduledTime: execution.ScheduledTime,

		Message: execution.Message,
		State:   execution.State,

		ExecutionStartTime: execution.ExecutionStartTime,
		ExecutionEndTime:   execution.ExecutionEndTime,
	}

	return output
}

func AsJobExecutionCollection(executions []*core.Execution) []*JobExecution {
	output := make([]*JobExecution, 0)

	for _, execution := range executions {
		output = append(output, AsJobExecution(execution))
	}

	return output
}

type JobExecution struct {
	GUID          string    `json:"guid"`
	JobGUID       string    `json:"job_guid"`
	TaskGUID      string    `json:"task_guid"`
	ScheduleGUID  string    `json:"schedule_guid,omitempty"`
	ScheduledTime time.Time `json:"scheduled_time,omitempty"`

	Message string `json:"message"`
	State   string `json:"state"`

	ExecutionStartTime time.Time `json:"execution_start_time"`
	ExecutionEndTime   time.Time `json:"execution_end_time"`
}
