package mock

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

type ScheduleService struct {
	storage []*core.Schedule
	jobs    core.JobService
	calls   core.CallService
	byApp   error
	locker  sync.Mutex
}

func NewScheduleService() *ScheduleService {
	return &ScheduleService{
		storage: make([]*core.Schedule, 0),
	}
}

func (service *ScheduleService) Persist(candidate *core.Schedule) (*core.Schedule, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	now := time.Now().UTC()

	id, err := uuid.NewRandom()

	if err != nil {
		return nil, fmt.Errorf("could not generate a job id")
	}

	candidate.GUID = id.String()
	candidate.CreatedAt = now
	candidate.UpdatedAt = now

	service.storage = append(service.storage, candidate)

	return candidate, nil
}

func (service *ScheduleService) ByCall(call *core.Call) ([]*core.Schedule, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	found := make([]*core.Schedule, 0)

	for _, candidate := range service.storage {
		if candidate.RefType == "call" && candidate.RefGUID == call.GUID {
			found = append(found, candidate)
		}
	}

	return found, nil
}

func (service *ScheduleService) ByJob(job *core.Job) ([]*core.Schedule, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	found := make([]*core.Schedule, 0)

	for _, candidate := range service.storage {
		if candidate.RefType == "job" && candidate.RefGUID == job.GUID {
			found = append(found, candidate)
		}
	}

	return found, nil
}

func (service *ScheduleService) Get(guid string) (*core.Schedule, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	candidates := make([]*core.Schedule, 0)

	for _, schedule := range service.storage {
		if schedule.GUID == guid {
			candidates = append(candidates, schedule)
		}
	}

	if len(candidates) > 1 {
		return nil, fmt.Errorf("too many results")
	}

	if len(candidates) < 1 {
		return nil, fmt.Errorf("too few results: %w", core.ErrNotFound)
	}

	return candidates[0], nil
}

func (service *ScheduleService) Delete(schedule *core.Schedule) error {
	service.locker.Lock()
	defer service.locker.Unlock()

	keep := make([]*core.Schedule, 0)
	for _, item := range service.storage {
		if item.GUID != schedule.GUID {
			keep = append(keep, item)
		}
	}

	service.storage = keep

	return nil
}

func (service *ScheduleService) Enabled() []*core.Schedule {
	service.locker.Lock()
	defer service.locker.Unlock()

	enabled := make([]*core.Schedule, 0)

	for _, candidate := range service.storage {
		if candidate.Enabled {
			enabled = append(enabled, candidate)
		}
	}

	return enabled
}

// Owners lets ByApp, Pause, and Resume find the app behind each schedule.
// Without it they match nothing.
func (service *ScheduleService) Owners(jobs core.JobService, calls core.CallService) {
	service.locker.Lock()
	defer service.locker.Unlock()

	service.jobs = jobs
	service.calls = calls
}

// ownedByApp must be called with locker held.
func (service *ScheduleService) ownedByApp(schedule *core.Schedule, appGUID string) bool {
	switch schedule.RefType {
	case "job":
		if service.jobs == nil {
			return false
		}
		job, err := service.jobs.Get(schedule.RefGUID)
		return err == nil && job.AppGUID == appGUID
	case "call":
		if service.calls == nil {
			return false
		}
		call, err := service.calls.Get(schedule.RefGUID)
		return err == nil && call.AppGUID == appGUID
	}

	return false
}

// FailByApp makes ByApp return err, as a lost database would. A nil err
// clears it.
func (service *ScheduleService) FailByApp(err error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	service.byApp = err
}

func (service *ScheduleService) ByApp(appGUID string) ([]*core.Schedule, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if service.byApp != nil {
		return nil, service.byApp
	}

	found := make([]*core.Schedule, 0)

	for _, candidate := range service.storage {
		if service.ownedByApp(candidate, appGUID) {
			found = append(found, candidate)
		}
	}

	return found, nil
}

// Orphaned needs Owners, and without it fails rather than call every
// schedule orphaned.
func (service *ScheduleService) Orphaned() ([]*core.Schedule, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if service.jobs == nil || service.calls == nil {
		return nil, fmt.Errorf("the mock schedule service has no owners to check")
	}

	found := make([]*core.Schedule, 0)

	for _, candidate := range service.storage {
		var err error
		switch candidate.RefType {
		case "job":
			_, err = service.jobs.Get(candidate.RefGUID)
		case "call":
			_, err = service.calls.Get(candidate.RefGUID)
		default:
			err = core.ErrNotFound
		}

		if errors.Is(err, core.ErrNotFound) {
			found = append(found, candidate)
		} else if err != nil {
			return nil, err
		}
	}

	return found, nil
}

func (service *ScheduleService) Pause(appGUID string, reason string) ([]*core.Schedule, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	changed := make([]*core.Schedule, 0)

	for _, candidate := range service.storage {
		if candidate.Enabled && service.ownedByApp(candidate, appGUID) {
			candidate.Enabled = false
			candidate.PausedBy = reason
			candidate.UpdatedAt = time.Now().UTC()
			changed = append(changed, candidate)
		}
	}

	return changed, nil
}

func (service *ScheduleService) Resume(appGUID string, reason string, guids []string) ([]*core.Schedule, error) {
	service.locker.Lock()
	defer service.locker.Unlock()

	changed := make([]*core.Schedule, 0)

	for _, candidate := range service.storage {
		if candidate.PausedBy == reason && len(reason) > 0 && slices.Contains(guids, candidate.GUID) && service.ownedByApp(candidate, appGUID) {
			candidate.Enabled = true
			candidate.PausedBy = ""
			candidate.UpdatedAt = time.Now().UTC()
			changed = append(changed, candidate)
		}
	}

	return changed, nil
}
