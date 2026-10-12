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
