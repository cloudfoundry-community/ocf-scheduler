package postgres

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

var instanceColumns = []string{
	"instance_id", // TEXT PRIMARY KEY,
	"service_id",  // TEXT NOT NULL,
	"plan_id",     // TEXT NOT NULL,
	"org_guid",    // TEXT,
	"space_guid",  // TEXT NOT NULL UNIQUE,
	"context",     // JSONB,
	"created_at",  // TIMESTAMP WITH TIME ZONE NOT NULL,
	"updated_at",  // TIMESTAMP WITH TIME ZONE NOT NULL
}

type InstanceService struct {
	db                  *sql.DB
	queryByPrimaryKey   string
	queryBySpace        string
	insertIntoInstances string
}

func NewInstanceService(db *sql.DB) *InstanceService {
	return &InstanceService{
		db:                  db,
		queryByPrimaryKey:   "SELECT " + strings.Join(instanceColumns, ", ") + " FROM service_instances WHERE instance_id = $1",
		queryBySpace:        "SELECT " + strings.Join(instanceColumns, ", ") + " FROM service_instances WHERE space_guid = $1",
		insertIntoInstances: generateInsert("service_instances", instanceColumns),
	}
}

func (service *InstanceService) Get(instanceID string) (*core.ServiceInstance, error) {
	return service.getOne(service.queryByPrimaryKey, instanceID)
}

func (service *InstanceService) InSpace(spaceGUID string) (*core.ServiceInstance, error) {
	return service.getOne(service.queryBySpace, spaceGUID)
}

func (service *InstanceService) Persist(candidate *core.ServiceInstance) (*core.ServiceInstance, error) {
	var context sql.NullString
	if candidate.Context != nil {
		encoded, err := json.Marshal(candidate.Context)
		if err != nil {
			return nil, fmt.Errorf("could not encode the instance context: %w", err)
		}
		context = sql.NullString{String: string(encoded), Valid: true}
	}

	now := time.Now().UTC()
	candidate.CreatedAt = now
	candidate.UpdatedAt = now

	err := WithTransaction(service.db, func(tx Transaction) error {
		_, iErr := tx.Exec(
			service.insertIntoInstances,
			candidate.InstanceID,
			candidate.ServiceID,
			candidate.PlanID,
			nullable(candidate.OrgGUID),
			candidate.SpaceGUID,
			context,
			candidate.CreatedAt,
			candidate.UpdatedAt,
		)

		return iErr
	})

	if err != nil {
		return nil, uniqueViolation(err)
	}

	return candidate, nil
}

func (service *InstanceService) Delete(instanceID string) error {
	return WithTransaction(service.db, func(tx Transaction) error {
		_, dErr := tx.Exec("DELETE FROM service_instances WHERE instance_id = $1", instanceID)
		return dErr
	})
}

func (service *InstanceService) getOne(query string, arg string) (*core.ServiceInstance, error) {
	var instance core.ServiceInstance
	var orgGUID sql.NullString
	var context []byte

	err := service.db.QueryRow(query, arg).Scan(
		&instance.InstanceID,
		&instance.ServiceID,
		&instance.PlanID,
		&orgGUID,
		&instance.SpaceGUID,
		&context,
		&instance.CreatedAt,
		&instance.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("service instance %w", core.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}

	instance.OrgGUID = orgGUID.String
	if len(context) > 0 {
		if err := json.Unmarshal(context, &instance.Context); err != nil {
			return nil, fmt.Errorf("could not decode the context of service instance %s: %w", instance.InstanceID, err)
		}
	}

	return &instance, nil
}

// uniqueViolation turns a Postgres unique violation into core.ErrConflict and
// passes any other error through.
func uniqueViolation(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return fmt.Errorf("%w: %s", core.ErrConflict, pqErr.Constraint)
	}

	return err
}
