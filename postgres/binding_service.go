package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

var bindingColumns = []string{
	"binding_id",  // TEXT PRIMARY KEY,
	"instance_id", // TEXT NOT NULL REFERENCES service_instances,
	"app_guid",    // TEXT,
	"created_at",  // TIMESTAMP WITH TIME ZONE NOT NULL
}

type BindingService struct {
	db                 *sql.DB
	queryByPrimaryKey  string
	queryByInstance    string
	insertIntoBindings string
}

func NewBindingService(db *sql.DB) *BindingService {
	return &BindingService{
		db:                 db,
		queryByPrimaryKey:  "SELECT " + strings.Join(bindingColumns, ", ") + " FROM service_bindings WHERE binding_id = $1",
		queryByInstance:    "SELECT " + strings.Join(bindingColumns, ", ") + " FROM service_bindings WHERE instance_id = $1 ORDER BY created_at ASC",
		insertIntoBindings: generateInsert("service_bindings", bindingColumns),
	}
}

func (service *BindingService) Get(bindingID string) (*core.ServiceBinding, error) {
	bindings, err := service.getCollection(service.queryByPrimaryKey, bindingID)
	if err != nil {
		return nil, err
	}

	if len(bindings) == 0 {
		return nil, fmt.Errorf("service binding %w", core.ErrNotFound)
	}

	return bindings[0], nil
}

func (service *BindingService) ForInstance(instanceID string) ([]*core.ServiceBinding, error) {
	return service.getCollection(service.queryByInstance, instanceID)
}

func (service *BindingService) Persist(candidate *core.ServiceBinding) (*core.ServiceBinding, error) {
	candidate.CreatedAt = time.Now().UTC()

	err := WithTransaction(service.db, func(tx Transaction) error {
		_, iErr := tx.Exec(
			service.insertIntoBindings,
			candidate.BindingID,
			candidate.InstanceID,
			nullable(candidate.AppGUID),
			candidate.CreatedAt,
		)

		return iErr
	})

	if err != nil {
		return nil, uniqueViolation(err)
	}

	return candidate, nil
}

func (service *BindingService) Delete(bindingID string) error {
	return WithTransaction(service.db, func(tx Transaction) error {
		_, dErr := tx.Exec("DELETE FROM service_bindings WHERE binding_id = $1", bindingID)
		return dErr
	})
}

func (service *BindingService) Bound(appGUID string) (bool, error) {
	if len(appGUID) == 0 {
		return false, nil
	}

	var bound bool
	err := service.db.QueryRow(
		"SELECT EXISTS (SELECT 1 FROM service_bindings WHERE app_guid = $1)",
		appGUID,
	).Scan(&bound)
	if err != nil {
		return false, err
	}

	return bound, nil
}

func (service *BindingService) getCollection(query string, arg string) ([]*core.ServiceBinding, error) {
	rows, err := service.db.Query(query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	collection := make([]*core.ServiceBinding, 0)

	for rows.Next() {
		var binding core.ServiceBinding
		var appGUID sql.NullString

		if err := rows.Scan(&binding.BindingID, &binding.InstanceID, &appGUID, &binding.CreatedAt); err != nil {
			return nil, err
		}

		binding.AppGUID = appGUID.String
		collection = append(collection, &binding)
	}

	if err := rows.Err(); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return collection, nil
}
