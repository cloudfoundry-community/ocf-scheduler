package mock

import (
	"fmt"
	"sync"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// InstanceService is an in-memory InstanceService that keeps the one
// instance per space rule the database enforces.
type InstanceService struct {
	storage map[string]*core.ServiceInstance
	fail    error
	locker  sync.Mutex
}

func NewInstanceService() *InstanceService {
	return &InstanceService{
		storage: make(map[string]*core.ServiceInstance),
	}
}

// Fail makes every call return err, as a lost database would. A nil err
// clears it.
func (service *InstanceService) Fail(err error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	service.fail = err
}

func (service *InstanceService) Get(instanceID string) (*core.ServiceInstance, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if service.fail != nil {
		return nil, service.fail
	}

	instance, found := service.storage[instanceID]
	if !found {
		return nil, fmt.Errorf("service instance %w", core.ErrNotFound)
	}

	copied := *instance
	return &copied, nil
}

func (service *InstanceService) InSpace(spaceGUID string) (*core.ServiceInstance, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if service.fail != nil {
		return nil, service.fail
	}

	for _, instance := range service.storage {
		if instance.SpaceGUID == spaceGUID {
			copied := *instance
			return &copied, nil
		}
	}

	return nil, fmt.Errorf("service instance %w", core.ErrNotFound)
}

func (service *InstanceService) Persist(candidate *core.ServiceInstance) (*core.ServiceInstance, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if service.fail != nil {
		return nil, service.fail
	}

	if _, found := service.storage[candidate.InstanceID]; found {
		return nil, fmt.Errorf("%w: instance id", core.ErrConflict)
	}

	for _, instance := range service.storage {
		if instance.SpaceGUID == candidate.SpaceGUID {
			return nil, fmt.Errorf("%w: space", core.ErrConflict)
		}
	}

	now := time.Now().UTC()
	candidate.CreatedAt = now
	candidate.UpdatedAt = now

	copied := *candidate
	service.storage[candidate.InstanceID] = &copied

	return candidate, nil
}

func (service *InstanceService) Delete(instanceID string) error {
	service.locker.Lock()
	defer service.locker.Unlock()

	if service.fail != nil {
		return service.fail
	}

	delete(service.storage, instanceID)

	return nil
}

// Count is the number of stored instances.
func (service *InstanceService) Count() int {
	service.locker.Lock()
	defer service.locker.Unlock()

	return len(service.storage)
}
