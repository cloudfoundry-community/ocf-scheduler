package core

import "fmt"

// JobRun is a struct that wraps a Job and makes it Runnable.
type JobRun struct {
	job      *Job
	schedule *Schedule
	services *Services
}

func NewJobRun(job *Job, schedule *Schedule, services *Services) *JobRun {
	return &JobRun{
		job,
		schedule,
		services,
	}
}

func (run *JobRun) Run() {
	execution := &Execution{
		RefGUID:      run.job.GUID,
		RefType:      "job",
		ScheduleGUID: run.schedule.GUID,
	}

	execution, err := run.services.Executions.Persist(execution)
	if err != nil {
		run.services.Logger.Warn("job-run", fmt.Sprintf("failed to persist execution for job %s: %v", run.job.GUID, err))
	}

	run.services.Runner.Execute(
		run.services,
		execution,
		run.job,
	)

}

func (run *JobRun) Services() *Services {
	return run.services
}

func (run *JobRun) Job() *Job {
	return run.job
}

func (run *JobRun) Call() *Call {
	return nil
}

func (run *JobRun) Schedule() *Schedule {
	return run.schedule
}
