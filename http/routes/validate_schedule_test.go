package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func TestValidateBareExpression(t *testing.T) {
	f := newFixture(t)
	rec := f.post("/schedules/validate", `{"expression": "*/15 * * * *"}`, "jeremy")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var got core.ScheduleAnalysis
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Valid || got.Description != "at minutes 0, 15, 30 and 45 of every hour" || len(got.NextRuns) != 5 || got.Ref != nil {
		t.Errorf("got %+v", got)
	}
}

// from replaces now as the reference time, so the run lists are fixed.
func TestValidateFrom(t *testing.T) {
	f := newFixture(t)
	body := `{"expression": "CRON_TZ=UTC 0 9 * * *", "next": 2, "prev": 1, "from": "2030-01-01T00:00:00Z"}`
	rec := f.post("/schedules/validate", body, "jeremy")
	var got core.ScheduleAnalysis
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := []time.Time{time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC), time.Date(2030, 1, 2, 9, 0, 0, 0, time.UTC)}
	if rec.Code != http.StatusOK || len(got.NextRuns) != 2 || !got.NextRuns[0].Equal(want[0]) || !got.NextRuns[1].Equal(want[1]) ||
		len(got.PrevRuns) != 1 || !got.PrevRuns[0].Equal(time.Date(2029, 12, 31, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("status %d, body %s", rec.Code, rec.Body)
	}
}

func TestValidateBadFromIs422(t *testing.T) {
	f := newFixture(t)
	rec := f.post("/schedules/validate", `{"expression": "0 9 * * *", "from": "next tuesday"}`, "jeremy")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status %d, body %s", rec.Code, rec.Body)
	}
}

func TestValidateInvalidExpressionIs200(t *testing.T) {
	f := newFixture(t)
	rec := f.post("/schedules/validate", `{"expression": "60 * * * *"}`, "jeremy")
	var got core.ScheduleAnalysis
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != http.StatusOK || got.Valid || got.Errors[0].Code != "parse_error" || got.Errors[0].Field != "minute" {
		t.Errorf("status %d, body %s", rec.Code, rec.Body)
	}
}

func TestValidateInJobContext(t *testing.T) {
	f := newFixture(t)
	body := `{"expression": "H(15-45) 2 * * *", "ref_type": "job", "ref_guid": "` + f.job.GUID + `", "next": 2}`
	rec := f.post("/schedules/validate", body, "jeremy")
	var got core.ScheduleAnalysis
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	want := f.services.Cron.Analyze("H(15-45) 2 * * *", f.job.GUID, 2, 0, time.Time{})
	if rec.Code != http.StatusOK || got.Ref == nil || got.Ref.Name != "backup" || !got.Illustrative ||
		got.HashedFields[0].Resolved != want.HashedFields[0].Resolved || len(got.NextRuns) != 2 {
		t.Errorf("status %d, body %s", rec.Code, rec.Body)
	}
}

func TestValidateStoredSchedules(t *testing.T) {
	f := newFixture(t)
	for _, expr := range []string{"0 2 * * *", "0 0 30 2 *"} {
		f.services.Schedules.Persist(&core.Schedule{Enabled: true, Expression: expr, RefGUID: f.job.GUID, RefType: "job"})
	}
	rec := f.post("/schedules/validate", `{"ref_type": "job", "ref_guid": "`+f.job.GUID+`", "next": 1}`, "jeremy")
	var got scheduleAnalysisCollection
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != http.StatusOK || len(got.Resources) != 2 {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	if !got.Resources[0].Valid || got.Resources[1].Valid || got.Resources[1].Errors[0].Code != "never_fires" ||
		got.Resources[0].ScheduleGUID == "" || got.Resources[0].Enabled == nil {
		t.Errorf("body %s", rec.Body)
	}
}

func TestValidateRequestErrors(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		body, token string
		status      int
	}{
		{`{"expression": "* * * * *"}`, "", http.StatusUnauthorized},
		{`{}`, "jeremy", http.StatusUnprocessableEntity},
		{`{"ref_type": "job"}`, "jeremy", http.StatusUnprocessableEntity},
		{`{"ref_type": "app", "ref_guid": "x"}`, "jeremy", http.StatusUnprocessableEntity},
		{`{"expression": "* * * * *", "next": 101}`, "jeremy", http.StatusUnprocessableEntity},
		{`{"expression": "* * * * *", "prev": -1}`, "jeremy", http.StatusUnprocessableEntity},
		{`{"ref_type": "job", "ref_guid": "nope"}`, "jeremy", http.StatusNotFound},
		{`{"ref_type": "call", "ref_guid": "nope"}`, "jeremy", http.StatusNotFound},
		{`not json`, "jeremy", http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		if rec := f.post("/schedules/validate", tt.body, tt.token); rec.Code != tt.status {
			t.Errorf("%s: status %d, want %d (%s)", tt.body, rec.Code, tt.status, rec.Body)
		}
	}
	rec := f.post("/schedules/validate", `{"ref_type": "job", "ref_guid": "nope"}`, "jeremy")
	if !strings.Contains(rec.Body.String(), `"ref_not_found"`) {
		t.Errorf("404 body %s lacks ref_not_found", rec.Body)
	}
}
