package routes

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

// An orphan is a job or call whose app Cloud Controller no longer knows,
// such as one left behind when its app or space was deleted.
type orphan struct {
	GUID      string           `json:"guid"`
	Type      string           `json:"type"`
	Name      string           `json:"name"`
	AppGUID   string           `json:"app_guid"`
	SpaceGUID string           `json:"space_guid"`
	Schedules []*core.Schedule `json:"schedules"`
}

// An orphan schedule is a schedule whose job or call is not stored, such as
// one left behind when a delete could not finish. Unlike a schedule shown
// under its job or call, it names the job or call it belonged to.
type orphanSchedule struct {
	GUID           string    `json:"guid"`
	Enabled        bool      `json:"enabled"`
	Expression     string    `json:"expression"`
	ExpressionType string    `json:"expression_type"`
	RefType        string    `json:"ref_type"`
	RefGUID        string    `json:"ref_guid"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type orphanCollection struct {
	Resources       []*orphan         `json:"resources"`
	OrphanSchedules []*orphanSchedule `json:"orphan_schedules"`
}

type orphanDeletion struct {
	GUIDs []string `json:"guids"`
}

// The result of deleting one named GUID.
const (
	orphanDeleted   = "deleted"
	orphanAppExists = "app_exists"
	// A schedule GUID whose job or call is still stored.
	orphanParentExists = "parent_exists"
	orphanNotFound     = "not_found"
	orphanFailed       = "failed"
)

type orphanResult struct {
	GUID    string `json:"guid"`
	Type    string `json:"type,omitempty"`
	Result  string `json:"result"`
	Message string `json:"message,omitempty"`
}

type orphanResults struct {
	Results []*orphanResult `json:"results"`
}

// appReaderScopes are the scopes that let the scheduler's own client read
// every app. Without one, Cloud Controller answers 404 for each app it hides
// from the client, which reads the same as a deleted app.
var appReaderScopes = []string{
	"cloud_controller.admin",
	"cloud_controller.admin_read_only",
	"cloud_controller.global_auditor",
}

// appLookupLimit bounds how many apps a list asks Cloud Controller about at
// once.
const appLookupLimit = 10

func Orphans(e *echo.Echo, services *core.Services) {
	ListOrphans(e, services)
	DeleteOrphans(e, services)
}

func ListOrphans(e *echo.Echo, services *core.Services) {
	// List the jobs and calls whose app is gone, across every space, and
	// the schedules whose job or call is gone
	// GET /orphans
	e.GET("/orphans", func(c echo.Context) error {
		tag := "list-orphans"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if status := requireAdmin(services, tag, auth); status != 0 {
			return c.JSON(status, "")
		}

		jobs, err := services.Jobs.All()
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("error retrieving jobs: %v", err))
			return c.JSON(http.StatusInternalServerError, "error retrieving jobs")
		}

		calls, err := services.Calls.All()
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("error retrieving calls: %v", err))
			return c.JSON(http.StatusInternalServerError, "error retrieving calls")
		}

		schedules, err := services.Schedules.Orphaned()
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("error retrieving schedules whose job or call is gone: %v", err))
			return c.JSON(http.StatusInternalServerError, "error retrieving schedules")
		}

		executables := make([]core.Executable, 0, len(jobs)+len(calls))
		for _, job := range jobs {
			executables = append(executables, job)
		}
		for _, call := range calls {
			executables = append(executables, call)
		}

		// Only the jobs and calls send the list to Cloud Controller, so a
		// list of orphan schedules alone works without the scope.
		if len(executables) > 0 {
			if status, message := requireAppReader(services, tag); status != 0 {
				return c.JSON(status, message)
			}
		}

		// Ask about each app once. Any answer but not-found fails the whole
		// request, so a partial list never passes for a complete one.
		gone, err := appsGone(c.Request().Context(), services, executables)
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("could not check %v with Cloud Controller", err))

			var lookup *appLookupError
			if errors.As(err, &lookup) {
				return c.JSON(http.StatusBadGateway, fmt.Sprintf("could not check app %s with Cloud Controller", lookup.appGUID))
			}
			return c.JSON(http.StatusBadGateway, "could not check apps with Cloud Controller")
		}

		output := &orphanCollection{
			Resources:       make([]*orphan, 0),
			OrphanSchedules: make([]*orphanSchedule, 0, len(schedules)),
		}
		for _, executable := range executables {
			owner := ownerOf(executable)
			if !gone[owner.AppGUID] {
				continue
			}

			schedules, err := schedulesOf(services, executable)
			if err != nil {
				services.Logger.Error(tag, fmt.Sprintf("failed to list schedules for %s %s: %v", executable.Type(), owner.GUID, err))
				return c.JSON(http.StatusInternalServerError, "error retrieving schedules")
			}

			output.Resources = append(output.Resources, &orphan{
				GUID:      owner.GUID,
				Type:      executable.Type(),
				Name:      owner.Name,
				AppGUID:   owner.AppGUID,
				SpaceGUID: owner.SpaceGUID,
				Schedules: schedules,
			})
		}

		slices.SortStableFunc(output.Resources, func(a, b *orphan) int {
			return cmp.Or(
				cmp.Compare(a.SpaceGUID, b.SpaceGUID),
				cmp.Compare(a.AppGUID, b.AppGUID),
				cmp.Compare(a.Type, b.Type),
				cmp.Compare(a.Name, b.Name),
			)
		})

		for _, schedule := range schedules {
			output.OrphanSchedules = append(output.OrphanSchedules, &orphanSchedule{
				GUID:           schedule.GUID,
				Enabled:        schedule.Enabled,
				Expression:     schedule.Expression,
				ExpressionType: schedule.ExpressionType,
				RefType:        schedule.RefType,
				RefGUID:        schedule.RefGUID,
				CreatedAt:      schedule.CreatedAt,
				UpdatedAt:      schedule.UpdatedAt,
			})
		}

		slices.SortStableFunc(output.OrphanSchedules, func(a, b *orphanSchedule) int {
			return cmp.Or(
				cmp.Compare(a.RefType, b.RefType),
				cmp.Compare(a.RefGUID, b.RefGUID),
				cmp.Compare(a.GUID, b.GUID),
			)
		})

		services.Logger.Info(tag, fmt.Sprintf(
			"found %d jobs and calls whose app is gone, and %d schedules whose job or call is gone",
			len(output.Resources), len(output.OrphanSchedules),
		))
		return c.JSON(http.StatusOK, output)
	})
}

func DeleteOrphans(e *echo.Echo, services *core.Services) {
	// Delete the named jobs and calls whose app is gone, and the named
	// schedules whose job or call is gone
	// POST /orphans/delete
	e.POST("/orphans/delete", func(c echo.Context) error {
		tag := "delete-orphans"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if status := requireAdmin(services, tag, auth); status != 0 {
			return c.JSON(status, "")
		}

		input := &orphanDeletion{}
		if err := c.Bind(input); err != nil {
			services.Logger.Error(tag, fmt.Sprintf("could not read the request: %v", err))
			return c.JSON(http.StatusBadRequest, requestError("invalid_request", "send a JSON object with a guids list"))
		}

		guids := make([]string, 0, len(input.GUIDs))
		for _, guid := range input.GUIDs {
			if !slices.Contains(guids, guid) {
				guids = append(guids, guid)
			}
		}

		if len(guids) == 0 {
			services.Logger.Error(tag, "got no GUIDs to delete")
			return c.JSON(http.StatusBadRequest, requestError("guids_required", "name at least one job, call, or schedule GUID to delete"))
		}

		if asksCloudController(services, guids) {
			if status, message := requireAppReader(services, tag); status != 0 {
				return c.JSON(status, message)
			}
		}

		output := &orphanResults{Results: make([]*orphanResult, 0, len(guids))}
		for _, guid := range guids {
			output.Results = append(output.Results, deleteOrphan(c.Request().Context(), services, tag, guid))
		}

		return c.JSON(http.StatusOK, output)
	})
}

// appLookupError names the app whose lookup failed a list.
type appLookupError struct {
	appGUID string
	err     error
}

func (e *appLookupError) Error() string {
	return fmt.Sprintf("app %s: %v", e.appGUID, e.err)
}

func (e *appLookupError) Unwrap() error {
	return e.err
}

// appsGone asks Cloud Controller about each executable's app once, with up
// to appLookupLimit lookups in flight, and reports which apps are gone. The
// first lookup that fails cancels the rest, and its appLookupError is
// returned.
func appsGone(ctx context.Context, services *core.Services, executables []core.Executable) (map[string]bool, error) {
	appGUIDs := make([]string, 0, len(executables))
	for _, executable := range executables {
		if appGUID := ownerOf(executable).AppGUID; !slices.Contains(appGUIDs, appGUID) {
			appGUIDs = append(appGUIDs, appGUID)
		}
	}

	exists := make([]bool, len(appGUIDs))

	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(appLookupLimit)

	for i, appGUID := range appGUIDs {
		if ctx.Err() != nil {
			break
		}

		group.Go(func() error {
			// A lookup that waited for a slot while another failed is not
			// worth starting.
			if err := ctx.Err(); err != nil {
				return &appLookupError{appGUID, err}
			}

			found, err := services.Info.AppExists(ctx, appGUID)
			if err != nil {
				return &appLookupError{appGUID, err}
			}

			exists[i] = found
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	// The loop stops early only when ctx ends, and then a lookup has
	// already returned an error.
	gone := make(map[string]bool, len(appGUIDs))
	for i, appGUID := range appGUIDs {
		gone[appGUID] = !exists[i]
	}

	return gone, nil
}

// asksCloudController reports whether deleting guids would ask Cloud
// Controller about an app, which only a job or call does. A GUID whose
// lookup fails counts, so the scope is checked whenever there is doubt.
func asksCloudController(services *core.Services, guids []string) bool {
	for _, guid := range guids {
		if _, err := findExecutable(services, guid); !errors.Is(err, core.ErrNotFound) {
			return true
		}
	}

	return false
}

// requireAppReader answers zero when the scheduler's own Cloud Controller
// token carries one of appReaderScopes. Otherwise it answers the status and
// message to refuse with, before anyone asks Cloud Controller about an app.
func requireAppReader(services *core.Services, tag string) (int, string) {
	scopes, err := services.Info.ClientScopes()
	if err != nil {
		services.Logger.Error(tag, fmt.Sprintf("could not read the scopes on the scheduler's Cloud Controller token: %v", err))
		return http.StatusBadGateway, "could not read the scopes on the scheduler's Cloud Controller token"
	}

	for _, scope := range appReaderScopes {
		if slices.Contains(scopes, scope) {
			return 0, ""
		}
	}

	message := "the scheduler's Cloud Controller client needs the cloud_controller.admin, cloud_controller.admin_read_only, or cloud_controller.global_auditor scope, because without one it cannot tell a deleted app from one it is not allowed to see"
	services.Logger.Error(tag, fmt.Sprintf("%s; its token carries %v", message, scopes))
	return http.StatusInternalServerError, message
}

// deleteOrphan deletes the job or call with guid, and its schedules, only
// when Cloud Controller says its app is not found. A guid that names no job
// or call is tried as an orphan schedule.
func deleteOrphan(ctx context.Context, services *core.Services, tag, guid string) *orphanResult {
	result := &orphanResult{GUID: guid}

	executable, err := findExecutable(services, guid)
	if errors.Is(err, core.ErrNotFound) {
		return deleteOrphanSchedule(services, tag, guid)
	}
	if err != nil {
		services.Logger.Error(tag, fmt.Sprintf("could not look up %s: %v", guid, err))
		result.Result = orphanFailed
		result.Message = "could not look up the job or call"
		return result
	}

	owner := ownerOf(executable)
	result.Type = executable.Type()

	exists, err := services.Info.AppExists(ctx, owner.AppGUID)
	if err != nil {
		services.Logger.Error(tag, fmt.Sprintf("could not check app %s of %s %s with Cloud Controller: %v", owner.AppGUID, result.Type, guid, err))
		result.Result = orphanFailed
		result.Message = fmt.Sprintf("could not check app %s with Cloud Controller", owner.AppGUID)
		return result
	}

	if exists {
		services.Logger.Warn(tag, fmt.Sprintf("refusing to delete %s %s, because app %s exists", result.Type, guid, owner.AppGUID))
		result.Result = orphanAppExists
		result.Message = fmt.Sprintf("app %s exists", owner.AppGUID)
		return result
	}

	// Delete as the job and call delete routes do, schedules first, each
	// through the workflow that keeps cron and the database in step.
	schedules, err := schedulesOf(services, executable)
	if err != nil {
		services.Logger.Error(tag, fmt.Sprintf("failed to list schedules for %s %s: %v", result.Type, guid, err))
		result.Result = orphanFailed
		result.Message = "could not list its schedules"
		return result
	}

	for _, schedule := range schedules {
		if err := workflows.DeletingASchedule(services, schedule, executable); err != nil {
			services.Logger.Error(tag, fmt.Sprintf("failed to delete schedule %s for %s %s: %v", schedule.GUID, result.Type, guid, err))
			result.Result = orphanFailed
			result.Message = fmt.Sprintf("could not delete schedule %s", schedule.GUID)
			return result
		}
	}

	if err := deleteExecutable(services, executable); errors.Is(err, core.ErrNotFound) {
		// Another request deleted it between our lookup and our delete.
		services.Logger.Warn(tag, fmt.Sprintf("%s %s was already gone", result.Type, guid))
		result.Result = orphanNotFound
		return result
	} else if err != nil {
		services.Logger.Error(tag, fmt.Sprintf("failed to delete %s %s: %v", result.Type, guid, err))
		result.Result = orphanFailed
		result.Message = fmt.Sprintf("could not delete the %s", result.Type)
		return result
	}

	result.Result = orphanDeleted
	left, err := sweepSchedules(services, tag, executable)
	if err != nil {
		services.Logger.Error(tag, err.Error())
	}
	switch {
	case left == 1:
		result.Message = "a schedule created during the delete was left behind, and it is listed as an orphan schedule"
	case left > 1:
		result.Message = fmt.Sprintf("%d schedules created during the delete were left behind, and they are listed as orphan schedules", left)
	case err != nil:
		result.Message = "could not check for schedules created during the delete, so look for them in the orphan schedules list"
	}

	services.Logger.Info(tag, fmt.Sprintf("deleted %s %s of app %s in space %s", result.Type, guid, owner.AppGUID, owner.SpaceGUID))
	return result
}

// deleteOrphanSchedule deletes the schedule with guid only when its job or
// call is not stored, through the workflow that keeps cron and the database
// in step.
func deleteOrphanSchedule(services *core.Services, tag, guid string) *orphanResult {
	result := &orphanResult{GUID: guid}

	schedule, err := services.Schedules.Get(guid)
	if errors.Is(err, core.ErrNotFound) {
		services.Logger.Warn(tag, fmt.Sprintf("no job, call, or schedule %s", guid))
		result.Result = orphanNotFound
		return result
	}
	if err != nil {
		services.Logger.Error(tag, fmt.Sprintf("could not look up schedule %s: %v", guid, err))
		result.Result = orphanFailed
		result.Message = "could not look up the schedule"
		return result
	}

	result.Type = "schedule"
	parent := parentOf(schedule)

	_, err = findOwner(services, parent)
	if err == nil {
		services.Logger.Warn(tag, fmt.Sprintf("refusing to delete schedule %s, because %s %s exists", guid, parent.Type(), schedule.RefGUID))
		result.Result = orphanParentExists
		result.Message = fmt.Sprintf("%s %s exists", parent.Type(), schedule.RefGUID)
		return result
	}
	if !errors.Is(err, core.ErrNotFound) {
		services.Logger.Error(tag, fmt.Sprintf("could not look up %s %s of schedule %s: %v", parent.Type(), schedule.RefGUID, guid, err))
		result.Result = orphanFailed
		result.Message = fmt.Sprintf("could not look up %s %s", parent.Type(), schedule.RefGUID)
		return result
	}

	if err := dropOrphanSchedule(services, schedule, parent); err != nil {
		services.Logger.Error(tag, fmt.Sprintf("failed to delete schedule %s of %s %s: %v", guid, parent.Type(), schedule.RefGUID, err))
		result.Result = orphanFailed
		result.Message = "could not delete the schedule"
		return result
	}

	services.Logger.Info(tag, fmt.Sprintf("deleted schedule %s of %s %s, which is gone", guid, parent.Type(), schedule.RefGUID))
	result.Result = orphanDeleted
	return result
}

// parentOf stands in for the job or call that a schedule names, so the
// schedule can be looked up against it and deleted through the workflow.
func parentOf(schedule *core.Schedule) core.Executable {
	if schedule.RefType == "call" {
		return &core.Call{GUID: schedule.RefGUID}
	}

	return &core.Job{GUID: schedule.RefGUID}
}

// owner holds the fields that jobs and calls share.
type owner struct {
	GUID, Name, AppGUID, SpaceGUID string
}

func ownerOf(executable core.Executable) owner {
	if job, err := executable.ToJob(); err == nil {
		return owner{job.GUID, job.Name, job.AppGUID, job.SpaceGUID}
	}

	call, _ := executable.ToCall()
	return owner{call.GUID, call.Name, call.AppGUID, call.SpaceGUID}
}

// findExecutable looks for a job with guid and then a call. Its error wraps
// core.ErrNotFound only when neither exists.
func findExecutable(services *core.Services, guid string) (core.Executable, error) {
	job, err := services.Jobs.Get(guid)
	if err == nil {
		return job, nil
	}
	if !errors.Is(err, core.ErrNotFound) {
		return nil, err
	}

	call, err := services.Calls.Get(guid)
	if err != nil {
		return nil, err
	}

	return call, nil
}

func schedulesOf(services *core.Services, executable core.Executable) ([]*core.Schedule, error) {
	if job, err := executable.ToJob(); err == nil {
		return services.Schedules.ByJob(job)
	}

	call, _ := executable.ToCall()
	return services.Schedules.ByCall(call)
}

func deleteExecutable(services *core.Services, executable core.Executable) error {
	if job, err := executable.ToJob(); err == nil {
		return services.Jobs.Delete(job)
	}

	call, _ := executable.ToCall()
	return services.Calls.Delete(call)
}
