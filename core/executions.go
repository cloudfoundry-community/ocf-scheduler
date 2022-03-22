package core

type Execution struct {
	GUID               string `json:"guid"`
	TaskGUID           string `json:"task_guid"`
	ScheduleGUID       string `json:"schedule_guid,omitempty"`
	ScheduledTime      string `json:"scheduled_time"`
	Message            string `json:"message"`
	State              string `json:"state"`
	ExecutionStartTime string `json:"execution_start_time"`
	ExecutionEndTime   string `json:" execution_end_time"`

	RefGUID string `json:"-"`
	RefType string `json:"-"`
}

type ExecutionService interface {
	Persist(*Execution) (*Execution, error)
	Start(*Execution) (*Execution, error)
	Success(*Execution) (*Execution, error)
	Fail(*Execution) (*Execution, error)
	UpdateMessage(*Execution, string) (*Execution, error)
	ByJob(*Job) []*Execution
}
