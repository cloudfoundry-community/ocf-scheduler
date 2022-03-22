package core

type Run struct {
	Job      *Job
	Schedule *Schedule
	Services *Services
}

func (run *Run) Run() {
	execution := &Execution{
		RefGUID:      run.Job.GUID,
		RefType:      "job",
		ScheduleGUID: run.Schedule.GUID,
	}

	execution, _ = run.Services.Executions.Persist(execution)

	run.Services.Runner.Execute(
		run.Services,
		execution,
		run.Job,
	)

}

type RunService interface {
	Execute(*Services, *Execution, *Job)
}
