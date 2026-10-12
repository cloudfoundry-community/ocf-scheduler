package broker_test

import (
	"errors"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

// racingInstances runs race once, the first time the broker looks up the
// instance in a space without holding the binding lock. It answers with what
// the store held before race ran, as a lookup that finished just before a
// concurrent provision committed would. Once failLocked is called, every
// lookup made under the lock fails.
type racingInstances struct {
	*mock.InstanceService
	lock *sync.RWMutex
	race func()

	locker    sync.Mutex
	raced     bool
	lockedErr error
}

func (instances *racingInstances) failLocked(err error) {
	instances.locker.Lock()
	defer instances.locker.Unlock()

	instances.lockedErr = err
}

func (instances *racingInstances) InSpace(spaceGUID string) (*core.ServiceInstance, error) {
	instance, err := instances.InstanceService.InSpace(spaceGUID)

	if !instances.lock.TryRLock() {
		instances.locker.Lock()
		lockedErr := instances.lockedErr
		instances.locker.Unlock()

		if lockedErr != nil {
			return nil, lockedErr
		}

		return instance, err
	}
	instances.lock.RUnlock()

	instances.locker.Lock()
	raced := instances.raced
	instances.raced = true
	instances.locker.Unlock()

	if !raced {
		instances.race()
	}

	return instance, err
}

func (f *fixture) racingInstances(race func()) *racingInstances {
	instances := &racingInstances{
		InstanceService: f.instances,
		lock:            &f.services.BindingLock,
		race:            race,
	}
	f.services.Instances = instances

	return instances
}

// A provision that loses the space to a concurrent provision answers 409,
// as it would for an instance that was there first, and leaves the winner
// alone.
func TestProvisionLosesARaceForTheSpaceWithAConflict(t *testing.T) {
	cases := map[string]struct {
		before func(*testing.T, *fixture)
	}{
		"empty space": {
			before: func(*testing.T, *fixture) {},
		},
		// Cloud Controller does not know instance-0, so the winner
		// replaces it.
		"orphan replaced by the winner": {
			before: func(t *testing.T, f *fixture) {
				expectStatus(t, "orphan", f.provision("instance-0", "space-a"), nethttp.StatusCreated)
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			c.before(t, f)
			f.cf.PrepareServiceInstance("instance-1")
			f.cf.PrepareServiceInstance("instance-2")

			var winner *httptest.ResponseRecorder
			f.racingInstances(func() { winner = f.provision("instance-2", "space-a") })

			loser := f.provision("instance-1", "space-a")

			expectStatus(t, "winner", winner, nethttp.StatusCreated)
			expectStatus(t, "loser", loser, nethttp.StatusConflict)
			if !strings.Contains(loser.Body.String(), "only one") {
				t.Errorf("want a description that names the one-per-space rule, got %s", loser.Body.String())
			}

			if _, err := f.instances.Get("instance-2"); err != nil {
				t.Errorf("the winner must stay, got %v", err)
			}
			if _, err := f.instances.Get("instance-1"); !errors.Is(err, core.ErrNotFound) {
				t.Errorf("the loser must not be stored, got %v", err)
			}
			if n := f.instances.Count(); n != 1 {
				t.Errorf("the store holds %d instances, want 1", n)
			}
		})
	}
}

// A store failure after the unlocked check is still an error to retry, never
// a conflict.
func TestProvisionRaceKeepsStoreErrors(t *testing.T) {
	f := newFixture(t)
	f.cf.PrepareServiceInstance("instance-1")
	f.racingInstances(func() { f.instances.Fail(fmt.Errorf("%s", databaseError)) })

	rec := f.provision("instance-1", "space-a")
	expectStatus(t, "provision", rec, nethttp.StatusInternalServerError)
	if strings.Contains(rec.Body.String(), "pq:") {
		t.Errorf("the description carries the database error: %s", rec.Body.String())
	}

	f.instances.Fail(nil)
	if n := f.instances.Count(); n != 0 {
		t.Errorf("the store holds %d instances, want 0", n)
	}
}

// A store failure when the space is looked up again under the lock, after a
// concurrent provision took it, is still an error to retry, never a
// conflict.
func TestProvisionRaceKeepsSpaceLookupErrors(t *testing.T) {
	f := newFixture(t)
	f.cf.PrepareServiceInstance("instance-1")
	f.cf.PrepareServiceInstance("instance-2")

	var winner *httptest.ResponseRecorder
	var instances *racingInstances
	instances = f.racingInstances(func() {
		winner = f.provision("instance-2", "space-a")
		instances.failLocked(fmt.Errorf("%s", databaseError))
	})

	loser := f.provision("instance-1", "space-a")

	expectStatus(t, "winner", winner, nethttp.StatusCreated)
	expectStatus(t, "loser", loser, nethttp.StatusInternalServerError)
	if strings.Contains(loser.Body.String(), "pq:") {
		t.Errorf("the description carries the database error: %s", loser.Body.String())
	}

	if _, err := f.instances.Get("instance-1"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("the loser must not be stored, got %v", err)
	}
	if n := f.instances.Count(); n != 1 {
		t.Errorf("the store holds %d instances, want 1", n)
	}
}

// Of many first provisions racing for one space, exactly one is created and
// every other one answers 409.
func TestConcurrentProvisionsInOneSpace(t *testing.T) {
	const racers = 8

	f := newFixture(t)
	for i := range racers {
		f.cf.PrepareServiceInstance(fmt.Sprintf("instance-%d", i))
	}

	codes := make([]int, racers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			codes[i] = f.provision(fmt.Sprintf("instance-%d", i), "space-a").Code
		}()
	}
	close(start)
	wg.Wait()

	created, conflicts := 0, 0
	for i, code := range codes {
		switch code {
		case nethttp.StatusCreated:
			created++
		case nethttp.StatusConflict:
			conflicts++
		default:
			t.Errorf("instance-%d: got %d, want 201 or 409", i, code)
		}
	}

	if created != 1 || conflicts != racers-1 {
		t.Errorf("got %d created and %d conflicts, want 1 and %d", created, conflicts, racers-1)
	}
	if n := f.instances.Count(); n != 1 {
		t.Errorf("the store holds %d instances, want 1", n)
	}
}
