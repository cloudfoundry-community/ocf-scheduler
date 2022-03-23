package core

type Runnable interface {
	Run()
	Services() *Services
	Job() *Job
	Call() *Call
	Schedule() *Schedule
}

type RunService interface {
	ExecuteJob(*Services, *Execution, *Job)
	ExecuteCall(*Services, *Execution, *Call)
}
