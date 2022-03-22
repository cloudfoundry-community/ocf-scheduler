package core

type RunService interface {
	Execute(*Services, *Execution, *Job)
}
