package core

import "context"

type CronService interface {
	Start()
	Stop() context.Context
	Add(*Run) error
	Delete(*Run) error
	Count() int
}

type Services struct {
	Environment EnvironmentInfoService
	Jobs        JobService
	Schedules   ScheduleService
	Workers     WorkerService
	Runner      RunService
	Executions  ExecutionService
	Cron        CronService
}
