package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/http"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func main() {
	jobs := mock.NewJobService()
	environment := mock.NewEnvironmentInfoService()
	schedules := mock.NewScheduleService()

	services := &core.Services{
		Jobs:        jobs,
		Environment: environment,
		Schedules:   schedules,
	}

	server := http.Server("0.0.0.0:8000", services)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println("stopping the server")
		}
	}()

	fmt.Println("Listening for connections on", server.Addr)

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		server.Close()
		fmt.Println(err)
		os.Exit(2)
	}
}
