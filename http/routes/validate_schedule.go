package routes

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

const maxRuns = 100

type scheduleAnalysisCollection struct {
	Pagination *pagination              `json:"pagination"`
	Ref        *core.Ref                `json:"ref"`
	Resources  []*core.ScheduleAnalysis `json:"resources"`
}

func ValidateSchedule(e *echo.Echo, services *core.Services) {
	// Validate a cron expression, alone or for a job or call, or re-check the
	// stored schedules of a job or call
	// POST /schedules/validate
	e.POST("/schedules/validate", func(c echo.Context) error {
		tag := "validate-schedule"
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			services.Logger.Error(tag, "authentication failed")
			return c.JSON(http.StatusUnauthorized, "")
		}

		input := &core.ValidateRequest{}
		if err := c.Bind(input); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, requestError("bad_request", "could not read the request: %v", err))
		}

		next := 5
		if input.Next != nil {
			next = *input.Next
		}
		var from time.Time
		if input.From != nil {
			from = *input.From
		}
		switch {
		case input.Expression == "" && input.RefGUID == "":
			return c.JSON(http.StatusUnprocessableEntity, requestError("bad_request", "give an expression, a job or call, or both"))
		case (input.RefType == "") != (input.RefGUID == ""):
			return c.JSON(http.StatusUnprocessableEntity, requestError("bad_request", "ref_type and ref_guid go together"))
		case input.RefType != "" && input.RefType != "job" && input.RefType != "call":
			return c.JSON(http.StatusUnprocessableEntity, requestError("bad_request", "ref_type must be job or call, not %q", input.RefType))
		case next < 0 || next > maxRuns || input.Prev < 0 || input.Prev > maxRuns:
			return c.JSON(http.StatusUnprocessableEntity, requestError("bad_request", "next and prev must be 0 to %d", maxRuns))
		}

		var ref *core.Ref
		var stored []*core.Schedule
		switch input.RefType {
		case "job":
			job, err := services.Jobs.Get(input.RefGUID)
			if err != nil {
				return c.JSON(http.StatusNotFound, requestError("ref_not_found", "job %s not found", input.RefGUID))
			}
			ref = &core.Ref{Type: "job", GUID: job.GUID, Name: job.Name}
			stored = services.Schedules.ByJob(job)
		case "call":
			call, err := services.Calls.Get(input.RefGUID)
			if err != nil {
				return c.JSON(http.StatusNotFound, requestError("ref_not_found", "call %s not found", input.RefGUID))
			}
			ref = &core.Ref{Type: "call", GUID: call.GUID, Name: call.Name}
			stored = services.Schedules.ByCall(call)
		}

		if input.Expression != "" {
			analysis := services.Cron.Analyze(input.Expression, input.RefGUID, next, input.Prev, from)
			analysis.Ref = ref
			return c.JSON(http.StatusOK, analysis)
		}

		resources := make([]*core.ScheduleAnalysis, 0, len(stored))
		for _, schedule := range stored {
			analysis := services.Cron.Analyze(schedule.Expression, input.RefGUID, next, input.Prev, from)
			analysis.Ref = ref
			analysis.ScheduleGUID = schedule.GUID
			analysis.Enabled = &schedule.Enabled
			resources = append(resources, &analysis)
		}

		return c.JSON(http.StatusOK, &scheduleAnalysisCollection{
			Ref:       ref,
			Resources: resources,
			Pagination: &pagination{
				TotalPages:   1,
				TotalResults: len(resources),
				First:        &pageref{Href: "first"},
				Last:         &pageref{Href: "last"},
				Next:         &pageref{Href: "next"},
				Previous:     &pageref{Href: "previous"},
			},
		})
	})
}
