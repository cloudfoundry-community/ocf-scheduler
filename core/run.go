package core

// Runnable is an interface that describes a process that can be ran.
type Runnable interface {
	Run()
	Services() *Services
	Job() *Job
	Call() *Call
	Schedule() *Schedule
}

// RunService is an interface that describes the entrypoint for executing
// an Executable.
type RunService interface {
	Execute(*Services, *Execution, Executable)
}
