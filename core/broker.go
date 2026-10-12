package core

import (
	"errors"
	"time"
)

// PausedByUnbind marks a schedule that the service broker disabled when its
// app lost its last binding. Binding the app again re-enables only the
// schedules carrying this mark.
const PausedByUnbind = "unbind"

var (
	// ErrNotFound means the service instance or binding does not exist.
	ErrNotFound = errors.New("not found")

	// ErrConflict means the record would break a uniqueness rule, such as a
	// second service instance in a space or a reused instance or binding ID.
	ErrConflict = errors.New("conflict")
)

// ServiceInstance records that the scheduler is enabled for a space. A space
// holds at most one.
type ServiceInstance struct {
	InstanceID string
	ServiceID  string
	PlanID     string
	OrgGUID    string
	SpaceGUID  string
	Context    map[string]any

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ServiceBinding records that an app may be scheduled. A binding with no
// AppGUID is a service key, which enables no app.
type ServiceBinding struct {
	BindingID  string
	InstanceID string
	AppGUID    string

	CreatedAt time.Time
}

// InstanceService stores service instances.
type InstanceService interface {
	// Get returns the instance with the given ID, or an error wrapping
	// ErrNotFound.
	Get(instanceID string) (*ServiceInstance, error)

	// InSpace returns the space's instance, or an error wrapping ErrNotFound.
	InSpace(spaceGUID string) (*ServiceInstance, error)

	// Persist stores a new instance. It returns an error wrapping ErrConflict
	// when the ID is taken or the space already has an instance.
	Persist(*ServiceInstance) (*ServiceInstance, error)

	// Delete removes the instance. Its bindings must be gone first.
	Delete(instanceID string) error
}

// BindingService stores service bindings.
type BindingService interface {
	// Get returns the binding with the given ID, or an error wrapping
	// ErrNotFound.
	Get(bindingID string) (*ServiceBinding, error)

	// ForInstance returns every binding under the instance.
	ForInstance(instanceID string) ([]*ServiceBinding, error)

	// Persist stores a new binding. It returns an error wrapping ErrConflict
	// when the ID is taken.
	Persist(*ServiceBinding) (*ServiceBinding, error)

	// Delete removes the binding.
	Delete(bindingID string) error

	// Bound reports whether any binding names the app.
	Bound(appGUID string) (bool, error)
}
