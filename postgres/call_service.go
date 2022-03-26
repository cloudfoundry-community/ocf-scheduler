package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

type CallService struct {
	db *sql.DB
}

func NewCallService(db *sql.DB) *CallService {
	return &CallService{db}
}

func (service *CallService) Get(guid string) (*core.Call, error) {
	candidates := service.getCollection(
		"select * from calls where guid = $1",
		guid,
	)

	if err := expectingOne(len(candidates)); err != nil {
		return nil, err
	}

	return candidates[0], nil
}

func (service *CallService) Delete(call *core.Call) error {
	// Let's not try to delete something that isn't in the db
	if _, err := service.Get(call.GUID); err != nil {
		return nil
	}

	err := WithTransaction(service.db, func(tx Transaction) error {
		_, dErr := tx.Exec(
			"DELETE FROM calls WHERE guid = $1",
			call.GUID,
		)

		return dErr
	})

	return err
}

func (service *CallService) Named(name string) (*core.Call, error) {
	candidates := service.getCollection(
		"select * from calls where name = $1",
		name,
	)

	if err := expectingOne(len(candidates)); err != nil {
		return nil, err
	}

	return candidates[0], nil
}

func (service *CallService) Persist(candidate *core.Call) (*core.Call, error) {
	now := time.Now().UTC()

	guid, err := core.GenGUID()
	if err != nil {
		return nil, fmt.Errorf("coult not generate a call id")
	}

	candidate.GUID = guid
	candidate.CreatedAt = now
	candidate.UpdatedAt = now

	err = WithTransaction(service.db, func(tx Transaction) error {
		_, aErr := tx.Exec(
			"INSERT INTO calls VALUES($1, $2, $3, $4, $5, $6, $7, $8)",
			candidate.GUID,
			candidate.Name,
			candidate.URL,
			candidate.AuthHeader,
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

func (service *CallService) InSpace(guid string) []*core.Call {
	return service.getCollection(
		"select * from calls where space_guid = $1 ORDER BY name ASC",
		guid,
	)
}

func (service *CallService) getCollection(query string, args ...interface{}) []*core.Call {
	collection := make([]*core.Call, 0)

	rows, err := service.db.Query(query, args...)
	if err != nil {
		return collection
	}

	for rows.Next() {
		var guid string
		var name string
		var url string
		var authHeader string
		var spaceGUID string
		var appGUID string
		var createdAt time.Time
		var updatedAt time.Time

		err := rows.Scan(&guid, &name, &url, &authHeader, &spaceGUID, &appGUID, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		candidate := &core.Call{
			GUID:       guid,
			Name:       name,
			URL:        url,
			AuthHeader: authHeader,
			SpaceGUID:  spaceGUID,
			AppGUID:    appGUID,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
		}

		collection = append(collection, candidate)
	}

	return collection
}
