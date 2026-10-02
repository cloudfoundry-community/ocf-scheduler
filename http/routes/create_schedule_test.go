package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

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
