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

	// A run with no stored execution has nothing to record its outcome on,
	// so it is skipped rather than run unrecorded.
	execution, err := run.services.Executions.Persist(execution)
	if err != nil {
		run.services.Logger.Error("job-run", fmt.Sprintf("skipping a run of job %s on schedule %s, because its execution could not be stored: %v", run.job.GUID, run.schedule.GUID, err))
		return
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
