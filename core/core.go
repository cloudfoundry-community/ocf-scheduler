package core

type Services struct {
	Environment EnvironmentInfoService
	Jobs        JobService
	Schedules   ScheduleService
	Workers     WorkerService
	Runner      RunService
	Executions  ExecutionService
}
