package postgres

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

type JobService struct {
	db                *sql.DB
	queryByPrimaryKey string
	queryByName       string
	queryBySpace      string
}

var jobColumns = []string{
	"guid",            // CHAR(36) PRIMARY KEY,
	"name",            // TEXT NOT NULL,
	"command",         // TEXT NOT NULL,
	"disk_in_mb",      // INT NOT NULL DEFAULT 1024,
	"memory_in_mb",    // INT NOT NULL DEFAULT 1024,
	"log_rate_in_bps", // INT NOT NULL DEFAULT 0,
	"state",           // TEXT NOT NULL,
	"app_guid",        // CHAR(36) NOT NULL,
	"space_guid",      // CHAR(36) NOT NULL,
	"created_at",      // TIMESTAMP WITH TIME ZONE NOT NULL,
	"updated_at",      // TIMESTAMP WITH TIME ZONE NOT NULL
}

const (
	JobStatePending   = "PENDING"
	JobStateSucceeded = "SUCCEEDED"
	JobStateFailed    = "FAILED"
)

func NewJobService(db *sql.DB) *JobService {
	qbpk := "SELECT " + strings.Join(jobColumns, ", ") + " FROM jobs WHERE guid = $1"
	qbn := "SELECT " + strings.Join(jobColumns, ", ") + " FROM jobs WHERE name = $1"
	qbs := "SELECT " + strings.Join(jobColumns, ", ") + " FROM jobs WHERE space_guid = $1 ORDER BY name ASC"
	return &JobService{db: db,
		queryByPrimaryKey: qbpk,
		queryByName:       qbn,
		queryBySpace:      qbs,
	}
}

func (service *JobService) Get(guid string) (*core.Job, error) {
	candidates, err := service.getCollection(
		service.queryByPrimaryKey, // "SELECT * FROM jobs WHERE guid = $1"
		guid,
	)
	if err != nil {
		return nil, err
	}

	if err := expectingOne(len(candidates)); err != nil {
		return nil, err
	}

	return candidates[0], nil
}

func (service *JobService) Delete(job *core.Job) error {
	// Let's not try to delete something that isn't in the db
	if _, err := service.Get(job.GUID); err != nil {
		return fmt.Errorf("job with GUID %s not found: %v", job.GUID, err)
	}

	err := WithTransaction(service.db, func(tx Transaction) error {
		_, dErr := tx.Exec(
			"DELETE FROM jobs WHERE guid = $1",
			job.GUID,
		)

		return dErr
	})

	return err
}

func (service *JobService) Named(name string) (*core.Job, error) {
	candidates, err := service.getCollection(
		service.queryByName, // "SELECT * FROM jobs WHERE name = $1"
		name,
	)
	if err != nil {
		return nil, err
	}

	if err := expectingOne(len(candidates)); err != nil {
		return nil, err
	}

	return candidates[0], nil
}

func (service *JobService) Persist(candidate *core.Job) (*core.Job, error) {
	now := time.Now().UTC()

	guid, err := core.GenGUID()
	if err != nil {
		return nil, fmt.Errorf("could not generate a job id: %v", err)
	}

	candidate.GUID = guid
	candidate.CreatedAt = now
	candidate.UpdatedAt = now
	candidate.State = JobStatePending

	if candidate.DiskInMb == 0 {
		candidate.DiskInMb = 1024
	}

	if candidate.MemoryInMb == 0 {
		candidate.MemoryInMb = 1024
	}

	err = WithTransaction(service.db, func(tx Transaction) error {
		_, aErr := tx.Exec(
			"INSERT INTO jobs (guid, name, command, disk_in_mb, memory_in_mb, log_rate_in_bps, state, app_guid, space_guid, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
			candidate.GUID,
			candidate.Name,
			candidate.Command,
			candidate.DiskInMb,
			candidate.MemoryInMb,
			candidate.LogRateInBps,
			candidate.State,
			candidate.AppGUID,
			candidate.SpaceGUID,
			candidate.CreatedAt,
			candidate.UpdatedAt,
		)

		return aErr
	})

	if err != nil {
		return nil, err
	}

	return candidate, nil
}

func (service *JobService) Success(candidate *core.Job) (*core.Job, error) {
	candidate.State = JobStateSucceeded
	return service.update(candidate)
}

func (service *JobService) Fail(candidate *core.Job) (*core.Job, error) {
	candidate.State = JobStateFailed
	return service.update(candidate)
}

func (service *JobService) update(candidate *core.Job) (*core.Job, error) {
	now := time.Now().UTC()

	candidate.UpdatedAt = now

	err := WithTransaction(service.db, func(tx Transaction) error {
		_, aErr := tx.Exec(
			"UPDATE jobs SET updated_at = $3, state = $2 WHERE guid = $1",
			candidate.GUID,
			candidate.State,
			candidate.UpdatedAt,
		)

		return aErr
	})

	if err != nil {
		return nil, err
	}

	return candidate, nil
}

func (service *JobService) InSpace(guid string) ([]*core.Job, error) {
	candidates, err := service.getCollection(
		service.queryBySpace, // "SELECT * FROM jobs WHERE space_guid = $1 ORDER BY name ASC",
		guid,
	)
	if err != nil {
		return nil, err
	}
	return candidates, nil
}

func (service *JobService) scanJob(rows *sql.Rows) (*core.Job, error) {
	var job core.Job
	err := rows.Scan(&job.GUID, &job.Name, &job.Command, &job.DiskInMb, &job.MemoryInMb, &job.LogRateInBps, &job.State, &job.AppGUID, &job.SpaceGUID, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (service *JobService) getCollection(query string, args ...any) ([]*core.Job, error) {
	var collection []*core.Job

	rows, err := service.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		job, err := service.scanJob(rows)
		if err != nil {
			return nil, err
		}
		collection = append(collection, job)
	}

	return collection, rows.Err()
}
