package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// refusingCron fails every Add, standing in for a cron that cannot take a
// schedule the create already stored.
type refusingCron struct{ core.CronService }

func (refusingCron) Add(core.Runnable) error {
	return errors.New("cron is full")
}

// A schedule that cron refuses is not left stored, where the next restart
// would load it into cron.
func TestCreateScheduleDeletesTheRowCronRefused(t *testing.T) {
	for _, tt := range []struct {
		name   string
		target func(tn *tenancy) core.Executable
	}{
		{name: "job", target: func(tn *tenancy) core.Executable { return tn.jobB }},
		{name: "call", target: func(tn *tenancy) core.Executable { return tn.callB }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tn := newTenancy(t)
			target := tt.target(tn)
			tn.services.Cron = refusingCron{tn.services.Cron}

			var stored *core.Schedule
			tn.services.Schedules = &capturingSchedules{tn.services.Schedules, &stored}

			rec := tn.do(http.MethodPost, "/"+tt.name+"s/"+ownerOf(target).GUID+"/schedules",
				`{"enabled": true, "expression": "0 3 * * *", "expression_type": "cron_expression"}`, tn.userB)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("status %d, want %d, body %s", rec.Code, http.StatusUnprocessableEntity, rec.Body)
			}

			if stored == nil {
				t.Fatal("the create never stored a schedule")
			}
			if _, err := tn.services.Schedules.Get(stored.GUID); !errors.Is(err, core.ErrNotFound) {
				t.Errorf("schedule %s is still stored (%v)", stored.GUID, err)
			}
		})
	}
}

func TestCreateScheduleReturnsStructuredErrors(t *testing.T) {
	f := newFixture(t)
	for _, path := range []string{"/jobs/" + f.job.GUID + "/schedules", "/calls/" + f.call.GUID + "/schedules"} {
		rec := f.post(path, `{"enabled": true, "expression": "60 * * * *", "expression_type": "cron_expression"}`, "jeremy")
		var got core.Findings
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusUnprocessableEntity ||
			got.Errors[0].Code != "parse_error" || got.Errors[0].ValidRange != "0-59" {
			t.Errorf("%s: status %d, body %s", path, rec.Code, rec.Body)
		}
		rec = f.post(path, `{"enabled": true, "expression": "0 0 30 2 *", "expression_type": "cron_expression"}`, "jeremy")
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "never_fires") {
			t.Errorf("%s never fires: status %d, body %s", path, rec.Code, rec.Body)
		}
		rec = f.post(path, `{"enabled": true, "expression": "H 2 * * *", "expression_type": "cron_expression"}`, "jeremy")
		if rec.Code != http.StatusCreated {
			t.Errorf("%s H: status %d, body %s", path, rec.Code, rec.Body)
		}
	}
}

func TestCreateScheduleStoresTrimmedExpression(t *testing.T) {
	f := newFixture(t)
	for _, path := range []string{"/jobs/" + f.job.GUID + "/schedules", "/calls/" + f.call.GUID + "/schedules"} {
		rec := f.post(path, `{"enabled": true, "expression": "  CRON_TZ=UTC 0 2 * * *  ", "expression_type": "cron_expression"}`, "jeremy")
		var got struct {
			Expression string `json:"expression"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusCreated ||
			got.Expression != "CRON_TZ=UTC 0 2 * * *" {
			t.Errorf("%s: status %d, expression %q, body %s", path, rec.Code, got.Expression, rec.Body)
		}
	}
}
