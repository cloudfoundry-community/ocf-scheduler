package presenters

import "github.com/starkandwayne/scheduler-for-ocf/core"

func AsCallExecution(execution *core.Execution) *CallExecution {
	output := &CallExecution{
		GUID:          execution.GUID,
		CallGUID:      execution.RefGUID,
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

func AsCallExecutionCollection(executions []*core.Execution) []*CallExecution {
	output := make([]*CallExecution, 0)

	for _, execution := range executions {
		output = append(output, AsCallExecution(execution))
	}

	return output
}

type CallExecution struct {
	GUID          string `json:"guid"`
	CallGUID      string `json:"call_guid"`
	TaskGUID      string `json:"task_guid"`
	ScheduleGUID  string `json:"schedule_guid,omitempty"`
	ScheduledTime string `json:"scheduled_time,omitempty"`

	Message string `json:"message"`
	State   string `json:"state"`

	ExecutionStartTime string `json:"execution_start_time"`
	ExecutionEndTime   string `json:"execution_end_time"`
}
