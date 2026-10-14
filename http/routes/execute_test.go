package routes

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// eagerRunner writes to the execution as soon as it is handed the run,
// standing in for a free worker that starts the run before the route has
// answered.
type eagerRunner struct{ core.RunService }

func (runner eagerRunner) Execute(services *core.Services, execution *core.Execution, executable core.Executable) {
	execution.State = "RUNNING"
	execution.Message = "the worker got here first"
	runner.RunService.Execute(services, execution, executable)
}

// A manual execute answers with the execution as it was stored, before the
// run was queued, so the answer never reads what the worker writes.
func TestExecuteAnswersWithTheExecutionBeforeTheRunStarts(t *testing.T) {
	for _, tt := range []struct {
		name   string
		target func(tn *tenancy) core.Executable
	}{
		{name: "job", target: func(tn *tenancy) core.Executable { return tn.jobB }},
		{name: "call", target: func(tn *tenancy) core.Executable { return tn.callB }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tn := newTenancy(t)
			tn.services.Runner = eagerRunner{tn.services.Runner}

			rec := tn.do(http.MethodPost, "/"+tt.name+"s/"+ownerOf(tt.target(tn)).GUID+"/execute", `{}`, tn.userB)
			if rec.Code != http.StatusCreated {
				t.Fatalf("status %d, want %d, body %s", rec.Code, http.StatusCreated, rec.Body)
			}

			var answer struct {
				State   string `json:"state"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &answer); err != nil {
				t.Fatalf("cannot read the answer %s: %v", rec.Body, err)
			}
			if answer.State == "RUNNING" || answer.Message != "" {
				t.Errorf("the answer has state %q and message %q, which the worker wrote after the execution was stored", answer.State, answer.Message)
			}
			if n := tn.runner.count(); n != 1 {
				t.Errorf("%d runs were dispatched, want 1", n)
			}
		})
	}
}
