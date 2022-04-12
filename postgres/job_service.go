package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

type JobService struct {
	db *sql.DB
}

func NewJobService(db *sql.DB) *JobService {
	return &JobService{db}
}

func (service *JobService) Get(guid string) (*core.Job, error) {
	candidates := service.getCollection(
		"select * from jobs where guid = $1",
		guid,
	)

	if err := expectingOne(len(candidates)); err != nil {
		return nil, err
	}

	return candidates[0], nil
}

func (service *JobService) Delete(job *core.Job) error {
	// Let's not try to delete something that isn't in the db
	if _, err := service.Get(job.GUID); err != nil {
		return nil
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
	candidates := service.getCollection(
		"select * from jobs where name = $1",
		name,
	)

	if err := expectingOne(len(candidates)); err != nil {
		return nil, err
	}

	return candidates[0], nil
}

func (service *JobService) Persist(candidate *core.Job) (*core.Job, error) {
	now := time.Now().UTC()

	guid, err := core.GenGUID()
	if err != nil {
		return nil, fmt.Errorf("coult not generate a job id")
	}

	candidate.GUID = guid
	candidate.CreatedAt = now
	candidate.UpdatedAt = now
	candidate.State = "PENDING"

	if candidate.DiskInMb == 0 {
		candidate.DiskInMb = 1024
	}

	if candidate.MemoryInMb == 0 {
		candidate.MemoryInMb = 1024
	}

	err = WithTransaction(service.db, func(tx Transaction) error {
		_, aErr := tx.Exec(
			"INSERT INTO jobs VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
			candidate.GUID,
			candidate.Name,
			candidate.Command,
			candidate.DiskInMb,
			candidate.MemoryInMb,
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
	candidate.State = "SUCCEEDED"

	return service.update(candidate)
}

func (service *JobService) Fail(candidate *core.Job) (*core.Job, error) {
	candidate.State = "FAILED"

	return service.update(candidate)
}

func (service *JobService) update(candidate *core.Job) (*core.Job, error) {
	now := time.Now().UTC()

	candidate.UpdatedAt = now

	err := WithTransaction(service.db, func(tx Transaction) error {
		_, aErr := tx.Exec(
			"update jobs set updated_at = $3, state = $2 where guid = $1",
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

func (service *JobService) InSpace(guid string) []*core.Job {
	return service.getCollection(
		"select * from jobs where space_guid = $1 ORDER BY name ASC",
		guid,
	)
}

func (service *JobService) getCollection(query string, args ...interface{}) []*core.Job {
	collection := make([]*core.Job, 0)

	rows, err := service.db.Query(query, args...)
	if err != nil {
		return collection
	}

	for rows.Next() {
		var guid string
		var name string
		var command string
		var diskInMb int
		var memoryInMb int
		var state string
		var spaceGUID string
		var appGUID string
		var createdAt time.Time
		var updatedAt time.Time

		err := rows.Scan(&guid, &name, &command, &diskInMb, &memoryInMb, &state, &appGUID, &spaceGUID, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		candidate := &core.Job{
			GUID:       guid,
			Name:       name,
			Command:    command,
			DiskInMb:   diskInMb,
			MemoryInMb: memoryInMb,
			State:      state,
			SpaceGUID:  spaceGUID,
			AppGUID:    appGUID,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
		}

		collection = append(collection, candidate)
	}

	return collection
}
