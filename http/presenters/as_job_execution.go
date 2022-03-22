package presenters

import "github.com/starkandwayne/scheduler-for-ocf/core"

func AsJobExecution(execution *core.Execution) *JobExecution {
	output := &JobExecution{
		GUID:     execution.GUID,
		JobGUID:  execution.RefGUID,
		TaskGUID: execution.TaskGUID,

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
	GUID     string `json:"guid"`
	JobGUID  string `json:"job_guid"`
	TaskGUID string `json:"task_guid"`

	Message string `json:"message"`
	State   string `json:"state"`

	ExecutionStartTime string `json:"execution_start_time"`
	ExecutionEndTime   string `json:"execution_end_time"`
}
