package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"

	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	cfconfig "github.com/cloudfoundry/go-cfclient/v3/config"
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

var SemVerMajor string
var SemVerMinor string
var SemVerPatch string
var SemVerPrerelease string
var SemVerBuild string
var BuildDate string
var BuildVcsUrl string
var BuildVcsId string
var BuildVcsIdDate string

var callRunner = http.NewRunService()

func ErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func main() {
	log := logger.New()
	tag := AppName
	bm := createBuildMeta(SemVerBuild)
	sv := createSemVer(SemVerMajor, SemVerMinor, SemVerPatch, SemVerPrerelease, bm)
	log.Info(tag, fmt.Sprintf("Version %s Build Date %s", sv, BuildDate))
	log.Info(tag, fmt.Sprintf("Vcs Info Url %s ID %s Date %s", BuildVcsUrl, BuildVcsId, BuildVcsIdDate))

	port := 8000
	portEnv := os.Getenv("SCHEDULER_PORT")
	if len(portEnv) > 0 {
		if t, err := strconv.Atoi(portEnv); err == nil {
			port = t
		}
	}

	clientID := os.Getenv("CLIENT_ID")
	if len(clientID) == 0 {
		log.Fatal(tag, "CLIENT_ID not set")
	}

	clientSecret := os.Getenv("CLIENT_SECRET")
	if len(clientSecret) == 0 {
		log.Fatal(tag, "CLIENT_SECRET not set")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if len(dbURL) == 0 {
		log.Fatal(tag, "DATABASE_URL not set")
	}

	cfEndpoint := os.Getenv("CF_ENDPOINT")
	if len(cfEndpoint) == 0 {
		log.Fatal(tag, "CF_ENDPOINT not set")
	}

	uaaEndpoint := os.Getenv("UAA_ENDPOINT")
	if len(uaaEndpoint) == 0 {
		log.Fatal(tag, "UAA_ENDPOINT not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		panic(fmt.Sprintf("could not open the database: %s", err.Error()))
	}
	defer db.Close()

	log.Debug(tag, fmt.Sprintf("migration collection size %d", len(migrations.Collection.Migrations)))

	n, err := migrate.Exec(db, "postgres", migrations.Collection, migrate.Up)
	if err != nil {
		log.Fatal(tag, fmt.Sprintf("could not update database schema: %s", err.Error()))
	}

	log.Info(tag, func(n int) string {
		phrase, plural := "database migration", "s"
		var count string

		if n == 0 {
			count = "no"
		} else {
			count = strconv.Itoa(n)
			if n == 1 {
				plural = ""
			}
		}
		return fmt.Sprintf("Applied %s %s%s", count, phrase, plural)
	}(n))

	log.Info(tag, "trying to instantiate a cf client")

	cfg, err := cfconfig.New(
		cfEndpoint,
		cfconfig.ClientCredentials(clientID, clientSecret),
		cfconfig.SkipTLSValidation(),
	)
	if err != nil {
		log.Fatal(tag, fmt.Sprintf("could not create cf config: %s", err.Error()))
	}

	rawClient, err := cfclient.New(cfg)
	if err != nil {
		log.Fatal(tag, fmt.Sprintf("could not instantiate cf client: %s", err.Error()))
	}

	cfClient := cf.NewRealCFClient(rawClient)

	log.Info(tag, "got the cf client set up")

	log.Info(tag, "trying to acquire the desired number of workers")

	workerNum := 20
	workerNumStr, exists := os.LookupEnv("SCHEDULER_WORKERS")
	if !exists {
		log.Info(tag, "No SCHEDULER_WORKERS provided, defaulting to 20")
	} else {
		workerNum, err = strconv.Atoi(workerNumStr)
		if err != nil || workerNum < 10 {
			log.Warn(tag, fmt.Sprintf("Invalid SCHEDULER_WORKERS value '%s': %v, defaulting to 20", workerNumStr, ErrorString(err)))
			workerNum = 20
		}
	}
	log.Info(tag, fmt.Sprintf("SCHEDULER_WORKERS set to %d", workerNum))

	timezonePath := filepath.Join(core.TimezoneJsonDir, core.TimezoneJsonBase)
	if err := cron.InitializeTimezones(timezonePath); err != nil {
		log.Warn(tag, fmt.Sprintf("Cannot process timezone file: %v", err.Error()))
		log.Warn(tag, "Starting scheduler without timezones loaded")
	} else {
		if serverTimezone, err := cron.GetServerTimezone(); err == nil {
			log.Info(tag, fmt.Sprintf("Sever timezone is %s", serverTimezone))
		}
		log.Info(tag, fmt.Sprintf("Timezone file loaded successfully with %d entries", len(cron.TimezonesSlice)))
	}

	auth := cf.NewAuthService(cfClient, log)
	jobs := postgres.NewJobService(db)
	calls := postgres.NewCallService(db)
	info := cf.NewInfoService(cfClient)
	jobRunner := cf.NewRunService(cfClient)
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
	defer func() {
		ctx := cronService.Stop()
		<-ctx.Done()
	}()

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
			} else {
				log.Warn(tag, fmt.Sprintf("skipping schedule %s: job %s not found", schedule.GUID, schedule.RefGUID))
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
			} else {
				log.Warn(tag, fmt.Sprintf("skipping schedule %s: call %s not found", schedule.GUID, schedule.RefGUID))
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
		log.Fatal(tag, fmt.Sprintf("server shutdown failed: %s", err.Error()))
	}
}
