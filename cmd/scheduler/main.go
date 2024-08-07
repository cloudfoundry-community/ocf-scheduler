package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	realcf "github.com/cloudfoundry-community/go-cfclient"
	"github.com/gammazero/workerpool"
	migrate "github.com/rubenv/sql-migrate"

	_ "github.com/lib/pq"

	"github.com/cloudfoundry-community/ocf-scheduler/cf"
	"github.com/cloudfoundry-community/ocf-scheduler/combined"
	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/cron"
	"github.com/cloudfoundry-community/ocf-scheduler/http"
	"github.com/cloudfoundry-community/ocf-scheduler/logger"
	"github.com/cloudfoundry-community/ocf-scheduler/postgres"
	"github.com/cloudfoundry-community/ocf-scheduler/postgres/migrations"
)

var AppName = "ocf-scheduler"
var Version = "(development)"

var callRunner = http.NewRunService()

func main() {
	log := logger.New()
	tag := AppName + " " + Version

	port := 8000
	portEnv := os.Getenv("SCHEDULER_PORT")
	if len(portEnv) > 0 {
		if t, err := strconv.Atoi(portEnv); err == nil {
			port = t
		}
	}

	clientID := os.Getenv("CLIENT_ID")
	if len(clientID) == 0 {
		log.Error(tag, "CLIENT_ID not set")
		os.Exit(255)
	}

	clientSecret := os.Getenv("CLIENT_SECRET")
	if len(clientSecret) == 0 {
		log.Error(tag, "CLIENT_SECRET not set")
		os.Exit(255)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if len(dbURL) == 0 {
		log.Error(tag, "DATABASE_URL not set")
		os.Exit(255)
	}

	cfEndpoint := os.Getenv("CF_ENDPOINT")
	if len(cfEndpoint) == 0 {
		log.Error(tag, "CF_ENDPOINT not set")
		os.Exit(255)
	}

	uaaEndpoint := os.Getenv("UAA_ENDPOINT")
	if len(uaaEndpoint) == 0 {
		log.Error(tag, "UAA_ENDPOINT not set")
		os.Exit(255)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		panic(fmt.Sprintf("could not open the database: %s", err.Error()))
	}
	defer db.Close()

	_, err = migrate.Exec(db, "postgres", migrations.Collection, migrate.Up)
	if err != nil {
		log.Error(tag, fmt.Sprintf("could not update database schema: %s", err.Error()))
		os.Exit(255)
	}

	//cfclient, err := mock.NewCFClient()
	cfg := &realcf.Config{
		ClientID:          clientID,
		ClientSecret:      clientSecret,
		ApiAddress:        cfEndpoint,
		SkipSslValidation: true,
	}

	log.Info(tag, "trying to instantiate a cf client")

	cfclient, err := realcf.NewClient(cfg)
	if err != nil {
		log.Error(tag, fmt.Sprintf("could not instantiate cf client: %s", err.Error()))
		os.Exit(255)
	}

	log.Info(tag, "got the cf client set up")

	log.Info(tag, "trying to acquire the desired number of workers")

	workerNumStr := os.Getenv("WORKER_NUM")
	if len(workerNumStr) == 0 {
		workerNumStr = "20"
		log.Info(tag, "No `WORKER_NUM` provided, defaulting to 20")
	} else {
		log.Info(tag, fmt.Sprintf("WORKER_NUM set to %s", workerNumStr))
	}

	workerNum, err := strconv.Atoi(workerNumStr)
	if err != nil {
		log.Error("Invalid WORKER_NUM: %v", err.Error())
		os.Exit(255)
	}

	auth := cf.NewAuthService(cfclient, log)
	jobs := postgres.NewJobService(db)
	calls := postgres.NewCallService(db)
	info := cf.NewInfoService(cfclient)
	jobRunner := cf.NewRunService(cfclient)
	schedules := postgres.NewScheduleService(db)
	executions := postgres.NewExecutionService(db)
	runner := combined.NewRunService(
		map[string]core.RunService{
			"job":  jobRunner,
			"call": callRunner,
		},
	)

	workers := workerpool.New(workerNum)
	defer workers.StopWait()

	cronService := cron.NewCronService(log)
	cronService.Start()
	defer cronService.Stop()

	services := &core.Services{
		Jobs:       jobs,
		Calls:      calls,
		Info:       info,
		Schedules:  schedules,
		Workers:    workers,
		Runner:     runner,
		Executions: executions,
		Cron:       cronService,
		Logger:     log,
		Auth:       auth,
	}

	// Load up all existing schedules
	log.Info(tag, "loading existing schedules")
	for _, schedule := range schedules.Enabled() {
		if schedule.RefType == "job" {
			if job, err := jobs.Get(schedule.RefGUID); err == nil {
				log.Info(
					tag,
					fmt.Sprintf(
						"loading job schedule for %s (%s)",
						job.Name,
						schedule.Expression,
					),
				)

				cronService.Add(core.NewJobRun(job, schedule, services))
			}
		} else {
			if call, err := calls.Get(schedule.RefGUID); err == nil {
				log.Info(
					tag,
					fmt.Sprintf(
						"loading call schedule for %s (%s)",
						call.Name,
						schedule.Expression,
					),
				)

				cronService.Add(core.NewCallRun(call, schedule, services))
			}
		}
	}

	// Schedule cleanup tasks
	retentionPeriod := 5 * 30 * 24 * time.Hour // 30 days
	ticker := time.NewTicker(24 * time.Hour)   // run cleanup every day
	quit := make(chan struct{})
	signalChan := make(chan os.Signal, 1) // New channel for OS signals

	go func() {
		for {
			select {
			case <-ticker.C:
				log.Info(tag, "Starting cleanup of old executions")

				deletedExecutions, err := executions.CleanupOldExecutions(retentionPeriod)
				if err != nil {
					log.Error(tag, fmt.Sprintf("Error cleaning up old executions: %s", err.Error()))
				} else {
					if len(deletedExecutions) == 0 {
						log.Info(tag, "No executions were deleted.")
					} else {
						for _, execution := range deletedExecutions {
							log.Info(tag, fmt.Sprintf("Deleted execution: %s, Last Updated: %s", execution["guid"], execution["execution_end_time"]))
						}
					}
				}

			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	server := http.Server(fmt.Sprintf("0.0.0.0:%d", port), services)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Info(tag, "stopping the server")
		}
	}()

	log.Info(tag, fmt.Sprintf("listening for connections on %s", server.Addr))

	signal.Notify(signalChan, os.Interrupt) // Use signalChan for signal notification

	<-signalChan // Wait for signal

	// Stop the cleanup ticker
	close(quit)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		server.Close()
		log.Error(tag, err.Error())
		os.Exit(2)
	}
}
