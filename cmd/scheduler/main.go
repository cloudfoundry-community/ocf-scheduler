package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/gammazero/workerpool"

	"github.com/starkandwayne/scheduler-for-ocf/combined"
	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/cron"
	"github.com/starkandwayne/scheduler-for-ocf/http"
	"github.com/starkandwayne/scheduler-for-ocf/logger"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

var callRunner = http.NewRunService()
var jobRunner = mock.NewRunService()

func main() {
	jobs := mock.NewJobService()
	calls := mock.NewCallService()
	environment := mock.NewEnvironmentInfoService()
	schedules := mock.NewScheduleService()
	executions := mock.NewExecutionService()
	log := logger.New()
	runner := combined.NewRunService(
		map[string]core.RunService{
			"job":  jobRunner,
			"call": callRunner,
		},
	)

	workers := workerpool.New(10)
	defer workers.StopWait()

	cronService := cron.NewCronService(log)
	cronService.Start()
	defer cronService.Stop()

	services := &core.Services{
		Jobs:        jobs,
		Calls:       calls,
		Environment: environment,
		Schedules:   schedules,
		Workers:     workers,
		Runner:      runner,
		Executions:  executions,
		Cron:        cronService,
		Logger:      log,
	}

	server := http.Server("0.0.0.0:8000", services)

	tag := "scheduler-for-ocf"

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Info(tag, "stopping the server")
		}
	}()

	log.Info(tag, fmt.Sprintf("listening for connections on %s", server.Addr))

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		server.Close()
		log.Error(tag, err.Error())
		os.Exit(2)
	}
}
