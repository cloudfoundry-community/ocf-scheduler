package routes

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/labstack/echo/v4"

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

type orphanCollection struct {
	Resources []*orphan `json:"resources"`
}

type orphanDeletion struct {
	GUIDs []string `json:"guids"`
}

// The result of deleting one named GUID.
const (
	orphanDeleted   = "deleted"
	orphanAppExists = "app_exists"
	orphanNotFound  = "not_found"
	orphanFailed    = "failed"
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

func Orphans(e *echo.Echo, services *core.Services) {
	ListOrphans(e, services)
	DeleteOrphans(e, services)
}

func ListOrphans(e *echo.Echo, services *core.Services) {
	// List the jobs and calls whose app is gone, across every space
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

		executables := make([]core.Executable, 0, len(jobs)+len(calls))
		for _, job := range jobs {
			executables = append(executables, job)
		}
		for _, call := range calls {
			executables = append(executables, call)
		}

		// Ask about each app once. Any answer but not-found fails the whole
		// request, so a partial list never passes for a complete one.
		gone := map[string]bool{}
		for _, executable := range executables {
			appGUID := ownerOf(executable).AppGUID
			if _, asked := gone[appGUID]; asked {
				continue
			}

			exists, err := services.Info.AppExists(c.Request().Context(), appGUID)
			if err != nil {
				services.Logger.Error(tag, fmt.Sprintf("could not check app %s with Cloud Controller: %v", appGUID, err))
				return c.JSON(http.StatusBadGateway, fmt.Sprintf("could not check app %s with Cloud Controller", appGUID))
			}

			gone[appGUID] = !exists
		}

		output := &orphanCollection{Resources: make([]*orphan, 0)}
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

		services.Logger.Info(tag, fmt.Sprintf("found %d jobs and calls whose app is gone", len(output.Resources)))
		return c.JSON(http.StatusOK, output)
	})
}

func DeleteOrphans(e *echo.Echo, services *core.Services) {
	// Delete the named jobs and calls whose app is gone
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
			return c.JSON(http.StatusBadRequest, requestError("guids_required", "name at least one job or call GUID to delete"))
		}

		output := &orphanResults{Results: make([]*orphanResult, 0, len(guids))}
		for _, guid := range guids {
			output.Results = append(output.Results, deleteOrphan(c.Request().Context(), services, tag, guid))
		}

		return c.JSON(http.StatusOK, output)
	})
}

// deleteOrphan deletes the job or call with guid, and its schedules, only
// when Cloud Controller says its app is not found.
func deleteOrphan(ctx context.Context, services *core.Services, tag, guid string) *orphanResult {
	result := &orphanResult{GUID: guid}

	executable, err := findExecutable(services, guid)
	if errors.Is(err, core.ErrNotFound) {
		services.Logger.Warn(tag, fmt.Sprintf("no job or call %s", guid))
		result.Result = orphanNotFound
		return result
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

	if err := deleteExecutable(services, executable); err != nil {
		services.Logger.Error(tag, fmt.Sprintf("failed to delete %s %s: %v", result.Type, guid, err))
		result.Result = orphanFailed
		result.Message = fmt.Sprintf("could not delete the %s", result.Type)
		return result
	}

	services.Logger.Info(tag, fmt.Sprintf("deleted %s %s of app %s in space %s", result.Type, guid, owner.AppGUID, owner.SpaceGUID))
	result.Result = orphanDeleted
	return result
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
