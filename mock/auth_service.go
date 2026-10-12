package mock

import (
	"fmt"
	"sync"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// AuthService is an in-memory AuthService keyed by the raw Authorization
// value. "jeremy" is a known caller with no spaces until Developer grants
// some.
type AuthService struct {
	admins      map[string]bool
	developers  map[string]map[string]bool
	lookupFails bool
	locker      sync.Mutex
}

func NewAuthService() *AuthService {
	service := &AuthService{
		admins:     make(map[string]bool),
		developers: make(map[string]map[string]bool),
	}

	service.Developer("jeremy")

	return service
}

// Admin makes auth a caller with cloud_controller.admin.
func (service *AuthService) Admin(auth string) {
	service.locker.Lock()
	defer service.locker.Unlock()

	service.admins[auth] = true
}

// Developer makes auth a known caller with space_developer in each space.
func (service *AuthService) Developer(auth string, spaces ...string) {
	service.locker.Lock()
	defer service.locker.Unlock()

	if _, found := service.developers[auth]; !found {
		service.developers[auth] = make(map[string]bool)
	}

	for _, space := range spaces {
		service.developers[auth][space] = true
	}
}

// FailLookups makes every role lookup fail, as an unreachable CF API would.
func (service *AuthService) FailLookups(fail bool) {
	service.locker.Lock()
	defer service.locker.Unlock()

	service.lookupFails = fail
}

func (service *AuthService) known(auth string) bool {
	_, developer := service.developers[auth]
	return developer || service.admins[auth]
}

func (service *AuthService) Verify(auth string) error {
	service.locker.Lock()
	defer service.locker.Unlock()

	if len(auth) == 0 {
		return fmt.Errorf("%w: no auth provided", core.ErrUnauthenticated)
	}

	if !service.known(auth) {
		return fmt.Errorf("%w: unknown caller", core.ErrUnauthenticated)
	}

	return nil
}

func (service *AuthService) VerifySpace(auth string, spaceGUID string) error {
	service.locker.Lock()
	defer service.locker.Unlock()

	if len(auth) == 0 || !service.known(auth) {
		return fmt.Errorf("%w: unknown caller", core.ErrUnauthenticated)
	}

	if service.admins[auth] {
		return nil
	}

	if service.lookupFails {
		return fmt.Errorf("%w: role lookup failed", core.ErrForbidden)
	}

	if len(spaceGUID) == 0 || !service.developers[auth][spaceGUID] {
		return fmt.Errorf("%w: not a space developer in %q", core.ErrForbidden, spaceGUID)
	}

	return nil
}
