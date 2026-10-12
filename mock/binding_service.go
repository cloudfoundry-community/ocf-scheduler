package mock

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// BindingService is an in-memory BindingService.
type BindingService struct {
	storage map[string]*core.ServiceBinding
	fail    error
	locker  sync.Mutex
}

func NewBindingService() *BindingService {
	return &BindingService{
		storage: make(map[string]*core.ServiceBinding),
	}
}

// Fail makes every call return err, as a lost database would. A nil err
// clears it.
func (service *BindingService) Fail(err error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	service.fail = err
}

func (service *BindingService) Get(bindingID string) (*core.ServiceBinding, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if service.fail != nil {
		return nil, service.fail
	}

	binding, found := service.storage[bindingID]
	if !found {
		return nil, fmt.Errorf("service binding %w", core.ErrNotFound)
	}

	copied := *binding
	return &copied, nil
}

func (service *BindingService) ForInstance(instanceID string) ([]*core.ServiceBinding, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if service.fail != nil {
		return nil, service.fail
	}

	found := make([]*core.ServiceBinding, 0)
	for _, binding := range service.storage {
		if binding.InstanceID == instanceID {
			copied := *binding
			found = append(found, &copied)
		}
	}

	sort.Slice(found, func(i, j int) bool { return found[i].BindingID < found[j].BindingID })

	return found, nil
}

func (service *BindingService) Persist(candidate *core.ServiceBinding) (*core.ServiceBinding, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if service.fail != nil {
		return nil, service.fail
	}

	if _, found := service.storage[candidate.BindingID]; found {
		return nil, fmt.Errorf("%w: binding id", core.ErrConflict)
	}

	candidate.CreatedAt = time.Now().UTC()

	copied := *candidate
	service.storage[candidate.BindingID] = &copied

	return candidate, nil
}

func (service *BindingService) Delete(bindingID string) error {
	service.locker.Lock()
	defer service.locker.Unlock()

	if service.fail != nil {
		return service.fail
	}

	delete(service.storage, bindingID)

	return nil
}

func (service *BindingService) Bound(appGUID string) (bool, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if service.fail != nil {
		return false, service.fail
	}

	if len(appGUID) == 0 {
		return false, nil
	}

	for _, binding := range service.storage {
		if binding.AppGUID == appGUID {
			return true, nil
		}
	}

	return false, nil
}

// Count is the number of stored bindings.
func (service *BindingService) Count() int {
	service.locker.Lock()
	defer service.locker.Unlock()

	return len(service.storage)
}
