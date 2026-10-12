package routes

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// authorize checks that the caller may act in spaceGUID, which needs
// cloud_controller.admin or a space_developer role in that space. It returns
// 0 when the caller may go on, or else the status to answer with. A caller
// whose token fails gets 401, and a caller without access gets denied. Routes
// that name a job or call by GUID pass 404 as denied, so the answer does not
// show whether another tenant's job or call exists. Routes that name an app
// or a space pass 403.
func authorize(services *core.Services, tag, auth, spaceGUID string, denied int) int {
	err := services.Auth.VerifySpace(auth, spaceGUID)
	if err == nil {
		return 0
	}

	if errors.Is(err, core.ErrUnauthenticated) {
		services.Logger.Error(tag, fmt.Sprintf("authentication failed: %v", err))
		return http.StatusUnauthorized
	}

	services.Logger.Error(tag, fmt.Sprintf("access to space %s denied: %v", spaceGUID, err))
	return denied
}

// missing picks the answer when the named job, call, schedule, or app is not
// there. An unauthenticated caller still gets 401, so the answer does not
// show whether the thing exists. Anyone else gets notFound.
func missing(services *core.Services, tag, auth string, notFound int) int {
	if err := services.Auth.Verify(auth); err != nil {
		services.Logger.Error(tag, fmt.Sprintf("authentication failed: %v", err))
		return http.StatusUnauthorized
	}

	return notFound
}

// ownedBy reports whether schedule belongs to the job or call with refGUID.
func ownedBy(schedule *core.Schedule, refType, refGUID string) bool {
	return schedule.RefType == refType && schedule.RefGUID == refGUID
}

// requireBinding enforces scheduler.broker.require_binding on the routes that
// create jobs, calls, and schedules or execute them. It runs after authorize,
// so only a caller with access to the space gets this far. When enforcement
// is on, the app must be bound to the space's scheduler service instance,
// and an admin may go on regardless. It returns 0 when the caller may go on,
// or else the status and body to answer with, and it reports whether the
// caller went on as an admin.
//
// It takes no lock, so it refuses an unbound app before the route reads the
// request body. The route then stores its work through whileBound, which
// checks the binding again under the lock.
func requireBinding(services *core.Services, tag, auth, appGUID string) (int, any, bool) {
	if !services.RequireBinding {
		return 0, nil, false
	}

	if services.Bindings == nil {
		services.Logger.Error(tag, "require_binding is on but the service broker is not running")
		return http.StatusInternalServerError, "", false
	}

	status, body := checkBound(services, tag, appGUID)
	if status == 0 {
		return 0, nil, false
	}

	if status == http.StatusForbidden && services.Auth.IsAdmin(auth) {
		return 0, nil, true
	}

	return status, body, false
}

// whileBound runs store, the step that stores a route's work and puts it in
// cron or dispatches it. When require_binding is on and the caller is not an
// admin, it holds services.BindingLock for reading, checks again that the
// app is bound, and runs store only if it is. The broker cannot unbind and
// pause the app in between, so work stored here is never left running for an
// app that has just been unbound. The lock covers nothing the client
// controls, because the route has read the body before it calls whileBound,
// and the lock is released before the route answers. It returns store's
// status and body, or the refusal.
func whileBound(services *core.Services, tag, appGUID string, admin bool, store func() (int, any)) (int, any) {
	if !services.RequireBinding || admin {
		return store()
	}

	services.BindingLock.RLock()
	defer services.BindingLock.RUnlock()

	if status, body := checkBound(services, tag, appGUID); status != 0 {
		return status, body
	}

	return store()
}

// checkBound returns 0 when the app is bound, 403 with a binding_required
// error when it is not, and 500 when the binding store fails.
func checkBound(services *core.Services, tag, appGUID string) (int, any) {
	bound, err := services.Bindings.Bound(appGUID)
	if err != nil {
		services.Logger.Error(tag, fmt.Sprintf("could not check the bindings of app %s: %v", appGUID, err))
		return http.StatusInternalServerError, ""
	}

	if bound {
		return 0, nil
	}

	services.Logger.Warn(tag, fmt.Sprintf("app %s is not bound to a scheduler service instance", appGUID))

	return http.StatusForbidden, requestError(
		"binding_required",
		"app %s must be bound to a scheduler service instance first; run cf bind-service with the app and the space's scheduler instance",
		appGUID,
	)
}
