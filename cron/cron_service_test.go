package cron

import (
	"testing"
	"time"

	cron "github.com/netresearch/go-cron"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/logger"
)

type stubRun struct{ schedule *core.Schedule }

func (r stubRun) Run()                     {}
func (r stubRun) Services() *core.Services { return nil }
func (r stubRun) Job() *core.Job           { return &core.Job{Name: "stub"} }
func (r stubRun) Call() *core.Call         { return nil }
func (r stubRun) Schedule() *core.Schedule { return r.schedule }

func TestAddRejectsOnlyParseErrors(t *testing.T) {
	service := NewCronService(logger.New(), Rules{MinInterval: time.Hour})
	ok := []string{"H(15-45) 2 * * *", "0 0 30 2 *", "*/5 * * * *"} // the last two break a policy rule
	for i, expr := range ok {
		s := &core.Schedule{GUID: string(rune('a' + i)), RefGUID: "job-guid-A", Expression: expr}
		if err := service.Add(stubRun{s}); err != nil {
			t.Errorf("%q: %v", expr, err)
		}
	}
	if err := service.Add(stubRun{&core.Schedule{GUID: "z", Expression: "60 * * * *"}}); err == nil {
		t.Error("60 * * * *: want an error")
	}
	if service.Count() != len(ok) {
		t.Errorf("registered %d entries, want %d", service.Count(), len(ok))
	}
}

// Validate ≡ create: Add registers the schedule Analyze computed the runs
// from, with the same H key.
func TestAddMatchesAnalyze(t *testing.T) {
	service := NewCronService(logger.New(), Rules{})
	s := &core.Schedule{GUID: "s1", RefGUID: "job-guid-A", Expression: "H(15-45) 2 * * MON-FRI"}
	if err := service.Add(stubRun{s}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	want := service.Analyze(s.Expression, s.RefGUID, 5, 0).NextRuns
	got := cron.NextN(service.Entry(service.mapping["s1"]).Schedule, now, 5)
	if len(want) != 5 || len(got) != len(want) {
		t.Fatalf("got %d registered and %d validated runs, want 5 of each", len(got), len(want))
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("run %d: registered %v, validated %v", i, got[i], want[i])
		}
	}
}
