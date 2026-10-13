package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

var callColumns = []string{
	"guid",        // CHAR(36) PRIMARY KEY,
	"name",        // TEXT NOT NULL,
	"url",         // TEXT NOT NULL,
	"auth_header", // TEXT NOT NULL,
	"app_guid",    // CHAR(36) NOT NULL,
	"space_guid",  // CHAR(36) NOT NULL,
	"created_at",  // TIMESTAMP WITH TIME ZONE NOT NULL,
	"updated_at",  // TIMESTAMP WITH TIME ZONE NOT NULL
}

type CallService struct {
	db                *sql.DB
	queryByPrimaryKey string
	queryByName       string
	queryBySpace      string
	queryAll          string
	insertIntoCalls   string
}

func NewCallService(db *sql.DB) *CallService {
	qbpk := "SELECT " + strings.Join(callColumns, ", ") + " FROM calls WHERE guid = $1"
	qbn := "SELECT " + strings.Join(callColumns, ", ") + " FROM calls WHERE name = $1"
	qbs := "SELECT " + strings.Join(callColumns, ", ") + " FROM calls WHERE space_guid = $1 ORDER BY name ASC"
	qa := "SELECT " + strings.Join(callColumns, ", ") + " FROM calls ORDER BY app_guid ASC, name ASC"
	is := generateInsert("calls", callColumns)
	return &CallService{db: db,
		queryByPrimaryKey: qbpk,
		queryByName:       qbn,
		queryBySpace:      qbs,
		queryAll:          qa,
		insertIntoCalls:   is,
	}
}

// Get reports a failed query, so a caller can tell a lost database from a
// call that does not exist.
func (service *CallService) Get(guid string) (*core.Call, error) {
	candidates, err := service.query(
		service.queryByPrimaryKey, // "select * from calls where guid = $1",
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

func (service *CallService) Delete(call *core.Call) error {
	// Let's not try to delete something that isn't in the db
	// A lookup that fails is not the same as a call that isn't there, so it
	// is returned.
	if _, err := service.Get(call.GUID); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil
		}

		return err
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

// toasted
func (service *CallService) Named(name string) (*core.Call, error) {
	candidates := service.getCollection(
		service.queryByName, // "select * from calls where name = $1",
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
			service.insertIntoCalls, // "INSERT INTO calls VALUES($1, $2, $3, $4, $5, $6, $7, $8)",
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

// toasted
func (service *CallService) InSpace(guid string) []*core.Call {
	return service.getCollection(
		service.queryBySpace, // "select * from calls where space_guid = $1 ORDER BY name ASC",
		guid,
	)
}

func (service *CallService) All() ([]*core.Call, error) {
	return service.query(
		service.queryAll, // "SELECT * FROM calls ORDER BY app_guid ASC, name ASC"
	)
}

func (service *CallService) getCollection(query string, args ...any) []*core.Call {
	collection := make([]*core.Call, 0)

	rows, err := service.db.Query(query, args...)
	if err != nil {
		return collection
	}

	for rows.Next() {
		candidate, err := scanCall(rows)
		if err != nil {
			continue
		}

		collection = append(collection, candidate)
	}

	return collection
}

// query is getCollection for callers that must not mistake a failed query
// or scan for a missing call.
func (service *CallService) query(query string, args ...any) ([]*core.Call, error) {
	collection := make([]*core.Call, 0)

	rows, err := service.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		candidate, err := scanCall(rows)
		if err != nil {
			return nil, err
		}

		collection = append(collection, candidate)
	}

	return collection, rows.Err()
}

func scanCall(rows *sql.Rows) (*core.Call, error) {
	var call core.Call

	err := rows.Scan(&call.GUID, &call.Name, &call.URL, &call.AuthHeader, &call.AppGUID, &call.SpaceGUID, &call.CreatedAt, &call.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &call, nil
}
