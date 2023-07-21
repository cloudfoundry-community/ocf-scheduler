package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

type ScheduleService struct {
	db *sql.DB
}

func NewScheduleService(db *sql.DB) *ScheduleService {
	return &ScheduleService{db}
}

func (service *ScheduleService) Get(guid string) (*core.Schedule, error) {
	candidates := service.getCollection(
		"select * from schedules where guid = $1",
		guid,
	)

	if err := expectingOne(len(candidates)); err != nil {
		return nil, err
	}

	return candidates[0], nil
}

func (service *ScheduleService) ByCall(call *core.Call) []*core.Schedule {
	return service.getCollection(
		"select * from schedules where ref_type = 'call' and ref_guid = $1",
		call.GUID,
	)
}

func (service *ScheduleService) Enabled() []*core.Schedule {
	return service.getCollection(
		"select * from schedules where enabled",
	)
}

func (service *ScheduleService) ByJob(job *core.Job) []*core.Schedule {
	return service.getCollection(
		"select * from schedules where ref_type = 'job' and ref_guid = $1",
		job.GUID,
	)
}

func (service *ScheduleService) Delete(schedule *core.Schedule) error {
	// Let's not try to delete something that isn't in the db
	if _, err := service.Get(schedule.GUID); err != nil {
		return nil
	}

	err := WithTransaction(service.db, func(tx Transaction) error {
		_, dErr := tx.Exec(
			"DELETE FROM schedules WHERE guid = $1",
			schedule.GUID,
		)

		return dErr
	})

	return err
}

func (service *ScheduleService) Persist(candidate *core.Schedule) (*core.Schedule, error) {
	now := time.Now().UTC()

	guid, err := core.GenGUID()
	if err != nil {
		return nil, fmt.Errorf("coult not generate a schedule id")
	}

	candidate.GUID = guid
	candidate.CreatedAt = now
	candidate.UpdatedAt = now

	err = WithTransaction(service.db, func(tx Transaction) error {
		_, aErr := tx.Exec(
			"INSERT INTO schedules VALUES($1, $2, $3, $4, $5, $6, $7, $8)",
			candidate.GUID,
			candidate.Enabled,
			candidate.Expression,
			candidate.ExpressionType,
			candidate.RefGUID,
			candidate.RefType,
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

func (service *ScheduleService) getCollection(query string, args ...interface{}) []*core.Schedule {
	collection := make([]*core.Schedule, 0)

	rows, err := service.db.Query(query, args...)
	if err != nil {
		return collection
	}

	for rows.Next() {
		var guid string
		var enabled bool
		var expression string
		var expressionType string
		var refGUID string
		var refType string
		var createdAt time.Time
		var updatedAt time.Time

		err := rows.Scan(&guid, &enabled, &expression, &expressionType, &refGUID, &refType, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		candidate := &core.Schedule{
			GUID:           guid,
			Enabled:        enabled,
			Expression:     expression,
			ExpressionType: expressionType,
			RefGUID:        refGUID,
			RefType:        refType,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
		}

		collection = append(collection, candidate)
	}

	return collection
}
