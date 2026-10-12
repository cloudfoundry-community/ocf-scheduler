package postgres

import (
	"errors"
	"testing"

	"github.com/ess/testscope"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/postgres/migrations"
)

func clearBrokerTables(t *testing.T) {
	t.Helper()

	for _, table := range []string{"service_bindings", "service_instances"} {
		if _, err := testdb.Exec("DELETE FROM " + table); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInstanceService(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	clearBrokerTables(t)
	t.Cleanup(func() { clearBrokerTables(t) })

	service := NewInstanceService(testdb)

	t.Run("a missing instance is not found", func(t *testing.T) {
		if _, err := service.Get("nope"); !errors.Is(err, core.ErrNotFound) {
			t.Errorf("want ErrNotFound, got %v", err)
		}
		if _, err := service.InSpace("space-nope"); !errors.Is(err, core.ErrNotFound) {
			t.Errorf("want ErrNotFound, got %v", err)
		}
	})

	stored, err := service.Persist(&core.ServiceInstance{
		InstanceID: "instance-1",
		ServiceID:  "service",
		PlanID:     "plan",
		OrgGUID:    "org-1",
		SpaceGUID:  "space-1",
		Context:    map[string]any{"platform": "cloudfoundry", "space_name": "dev"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if stored.CreatedAt.IsZero() {
		t.Error("Persist should set created_at")
	}

	t.Run("it reads back by ID and by space", func(t *testing.T) {
		for name, read := range map[string]func() (*core.ServiceInstance, error){
			"by id":    func() (*core.ServiceInstance, error) { return service.Get("instance-1") },
			"by space": func() (*core.ServiceInstance, error) { return service.InSpace("space-1") },
		} {
			instance, err := read()
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if instance.InstanceID != "instance-1" || instance.OrgGUID != "org-1" || instance.SpaceGUID != "space-1" ||
				instance.Context["space_name"] != "dev" {
				t.Errorf("%s: unexpected instance %+v", name, instance)
			}
		}
	})

	t.Run("a second instance in the space conflicts", func(t *testing.T) {
		_, err := service.Persist(&core.ServiceInstance{InstanceID: "instance-2", ServiceID: "service", PlanID: "plan", SpaceGUID: "space-1"})
		if !errors.Is(err, core.ErrConflict) {
			t.Errorf("want ErrConflict, got %v", err)
		}
	})

	t.Run("a reused ID conflicts", func(t *testing.T) {
		_, err := service.Persist(&core.ServiceInstance{InstanceID: "instance-1", ServiceID: "service", PlanID: "plan", SpaceGUID: "space-2"})
		if !errors.Is(err, core.ErrConflict) {
			t.Errorf("want ErrConflict, got %v", err)
		}
	})

	t.Run("an instance without an org or context reads back empty", func(t *testing.T) {
		if _, err := service.Persist(&core.ServiceInstance{InstanceID: "instance-3", ServiceID: "service", PlanID: "plan", SpaceGUID: "space-3"}); err != nil {
			t.Fatal(err)
		}
		instance, err := service.Get("instance-3")
		if err != nil {
			t.Fatal(err)
		}
		if instance.OrgGUID != "" || instance.Context != nil {
			t.Errorf("unexpected instance %+v", instance)
		}
	})

	t.Run("delete removes it", func(t *testing.T) {
		if err := service.Delete("instance-1"); err != nil {
			t.Fatal(err)
		}
		if _, err := service.Get("instance-1"); !errors.Is(err, core.ErrNotFound) {
			t.Errorf("want ErrNotFound, got %v", err)
		}
	})
}

func TestBindingService(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	clearBrokerTables(t)
	t.Cleanup(func() { clearBrokerTables(t) })

	instances := NewInstanceService(testdb)
	service := NewBindingService(testdb)

	if _, err := instances.Persist(&core.ServiceInstance{InstanceID: "instance-1", ServiceID: "service", PlanID: "plan", SpaceGUID: "space-1"}); err != nil {
		t.Fatal(err)
	}

	for _, binding := range []*core.ServiceBinding{
		{BindingID: "binding-1", InstanceID: "instance-1", AppGUID: "app-1"},
		{BindingID: "binding-2", InstanceID: "instance-1", AppGUID: "app-1"},
		{BindingID: "key-1", InstanceID: "instance-1"},
	} {
		if _, err := service.Persist(binding); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("a reused ID conflicts", func(t *testing.T) {
		_, err := service.Persist(&core.ServiceBinding{BindingID: "binding-1", InstanceID: "instance-1", AppGUID: "app-2"})
		if !errors.Is(err, core.ErrConflict) {
			t.Errorf("want ErrConflict, got %v", err)
		}
	})

	t.Run("a binding needs its instance", func(t *testing.T) {
		if _, err := service.Persist(&core.ServiceBinding{BindingID: "binding-9", InstanceID: "nope", AppGUID: "app-9"}); err == nil {
			t.Error("want an error for a missing instance")
		}
	})

	t.Run("it reads back", func(t *testing.T) {
		binding, err := service.Get("binding-1")
		if err != nil || binding.AppGUID != "app-1" || binding.InstanceID != "instance-1" {
			t.Errorf("unexpected binding %+v, %v", binding, err)
		}

		key, err := service.Get("key-1")
		if err != nil || key.AppGUID != "" {
			t.Errorf("unexpected service key %+v, %v", key, err)
		}

		if _, err := service.Get("nope"); !errors.Is(err, core.ErrNotFound) {
			t.Errorf("want ErrNotFound, got %v", err)
		}
	})

	t.Run("it lists the instance's bindings", func(t *testing.T) {
		bindings, err := service.ForInstance("instance-1")
		if err != nil || len(bindings) != 3 {
			t.Errorf("want 3 bindings, got %d, %v", len(bindings), err)
		}
	})

	t.Run("bound follows the app's last binding", func(t *testing.T) {
		expectBound := func(app string, want bool) {
			t.Helper()
			bound, err := service.Bound(app)
			if err != nil || bound != want {
				t.Errorf("Bound(%q) = %t, %v; want %t", app, bound, err, want)
			}
		}

		expectBound("app-1", true)
		expectBound("", false)
		expectBound("app-2", false)

		for _, check := range []struct {
			except string
			want   bool
		}{{"binding-1", true}, {"nope", true}} {
			if bound, err := service.BoundExcept("app-1", check.except); err != nil || bound != check.want {
				t.Errorf("BoundExcept(app-1, %s) = %t, %v; want %t", check.except, bound, err, check.want)
			}
		}

		if err := service.Delete("binding-1"); err != nil {
			t.Fatal(err)
		}
		expectBound("app-1", true)

		if bound, err := service.BoundExcept("app-1", "binding-2"); err != nil || bound {
			t.Errorf("BoundExcept(app-1, binding-2) = %t, %v; want false", bound, err)
		}

		if err := service.Delete("binding-2"); err != nil {
			t.Fatal(err)
		}
		expectBound("app-1", false)
	})
}

func TestScheduleServicePauseAndResume(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	Cleaner.Acquire("jobs", "calls", "schedules")
	defer Cleaner.Clean("jobs", "calls", "schedules")

	jobs := NewJobService(testdb)
	calls := NewCallService(testdb)
	schedules := NewScheduleService(testdb)

	job, err := jobs.Persist(&core.Job{Name: "backup", Command: "true", AppGUID: "app-1", SpaceGUID: "space-1"})
	if err != nil {
		t.Fatal(err)
	}
	call, err := calls.Persist(&core.Call{Name: "ping", URL: "https://example.com", AuthHeader: "x", AppGUID: "app-1", SpaceGUID: "space-1"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := jobs.Persist(&core.Job{Name: "other", Command: "true", AppGUID: "app-2", SpaceGUID: "space-1"})
	if err != nil {
		t.Fatal(err)
	}

	persist := func(ref string, refType string, enabled bool) *core.Schedule {
		t.Helper()
		schedule, err := schedules.Persist(&core.Schedule{Enabled: enabled, Expression: "0 2 * * *", ExpressionType: "cron_expression", RefGUID: ref, RefType: refType})
		if err != nil {
			t.Fatal(err)
		}
		return schedule
	}

	nightly := persist(job.GUID, "job", true)
	owned := persist(job.GUID, "job", false)
	ping := persist(call.GUID, "call", true)
	elsewhere := persist(other.GUID, "job", true)

	state := func(schedule *core.Schedule) *core.Schedule {
		t.Helper()
		stored, err := schedules.Get(schedule.GUID)
		if err != nil {
			t.Fatal(err)
		}
		return stored
	}

	if found, err := schedules.ByApp("app-1"); err != nil || len(found) != 3 {
		t.Errorf("ByApp found %d schedules, %v; want 3", len(found), err)
	}

	paused, err := schedules.Pause("app-1", core.PausedByUnbind)
	if err != nil {
		t.Fatal(err)
	}
	if len(paused) != 2 {
		t.Errorf("paused %d schedules, want 2", len(paused))
	}
	for _, schedule := range paused {
		if schedule.Enabled || schedule.PausedBy != core.PausedByUnbind {
			t.Errorf("Pause returned %+v", schedule)
		}
	}

	for name, schedule := range map[string]*core.Schedule{"nightly": nightly, "ping": ping} {
		if stored := state(schedule); stored.Enabled || stored.PausedBy != core.PausedByUnbind {
			t.Errorf("%s: want paused by unbind, got enabled=%t paused_by=%q", name, stored.Enabled, stored.PausedBy)
		}
	}
	if stored := state(owned); stored.Enabled || stored.PausedBy != "" {
		t.Errorf("owned: want disabled and unmarked, got enabled=%t paused_by=%q", stored.Enabled, stored.PausedBy)
	}
	if stored := state(elsewhere); !stored.Enabled {
		t.Error("another app's schedule was paused")
	}

	again, err := schedules.Pause("app-1", core.PausedByUnbind)
	if err != nil || len(again) != 0 {
		t.Errorf("pausing again changed %d schedules, %v", len(again), err)
	}

	// Resume changes only the named schedules of the app that still carry
	// the mark.
	resumed, err := schedules.Resume("app-1", core.PausedByUnbind, []string{nightly.GUID, owned.GUID, elsewhere.GUID})
	if err != nil {
		t.Fatal(err)
	}
	if len(resumed) != 1 || resumed[0].GUID != nightly.GUID {
		t.Errorf("resumed %d schedules, want only nightly", len(resumed))
	}
	if stored := state(ping); stored.Enabled || stored.PausedBy != core.PausedByUnbind {
		t.Errorf("ping: resume changed a schedule it was not given, got enabled=%t paused_by=%q", stored.Enabled, stored.PausedBy)
	}

	resumed, err = schedules.Resume("app-1", core.PausedByUnbind, []string{ping.GUID})
	if err != nil {
		t.Fatal(err)
	}
	if len(resumed) != 1 {
		t.Errorf("resumed %d schedules, want 1", len(resumed))
	}

	for name, schedule := range map[string]*core.Schedule{"nightly": nightly, "ping": ping} {
		if stored := state(schedule); !stored.Enabled || stored.PausedBy != "" {
			t.Errorf("%s: want enabled and unmarked, got enabled=%t paused_by=%q", name, stored.Enabled, stored.PausedBy)
		}
	}
	if stored := state(owned); stored.Enabled {
		t.Error("resume enabled a schedule its owner disabled")
	}
}

// The broker migrations go down and back up cleanly.
func TestBrokerMigrationsRoundTrip(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	clearBrokerTables(t)

	exists := func(query string, args ...any) bool {
		t.Helper()
		var found bool
		if err := testdb.QueryRow(query, args...).Scan(&found); err != nil {
			t.Fatal(err)
		}
		return found
	}
	table := func(name string) bool {
		return exists("SELECT to_regclass($1) IS NOT NULL", name)
	}
	column := func() bool {
		return exists("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'schedules' AND column_name = 'paused_by')")
	}

	if !table("service_instances") || !table("service_bindings") || !column() {
		t.Fatal("the broker schema should be in place before the test")
	}

	if n, err := migrate.ExecMax(testdb, "postgres", migrations.Collection, migrate.Down, 3); err != nil || n != 3 {
		t.Fatalf("down: applied %d, %v", n, err)
	}
	if table("service_instances") || table("service_bindings") || column() {
		t.Error("down should remove the broker tables and the paused_by column")
	}

	if n, err := migrate.Exec(testdb, "postgres", migrations.Collection, migrate.Up); err != nil || n != 3 {
		t.Fatalf("up: applied %d, %v", n, err)
	}
	if !table("service_instances") || !table("service_bindings") || !column() {
		t.Error("up should restore the broker schema")
	}
}
