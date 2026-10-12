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
// or else the status and body to answer with.
func requireBinding(services *core.Services, tag, auth, appGUID string) (int, any) {
	if !services.RequireBinding {
		return 0, nil
	}

	if services.Bindings == nil {
		services.Logger.Error(tag, "require_binding is on but the service broker is not running")
		return http.StatusInternalServerError, ""
	}

	bound, err := services.Bindings.Bound(appGUID)
	if err != nil {
		services.Logger.Error(tag, fmt.Sprintf("could not check the bindings of app %s: %v", appGUID, err))
		return http.StatusInternalServerError, ""
	}

	if bound || services.Auth.IsAdmin(auth) {
		return 0, nil
	}

	services.Logger.Warn(tag, fmt.Sprintf("app %s is not bound to a scheduler service instance", appGUID))

	return http.StatusForbidden, requestError(
		"binding_required",
		"app %s must be bound to a scheduler service instance first; run cf bind-service with the app and the space's scheduler instance",
		appGUID,
	)
}
