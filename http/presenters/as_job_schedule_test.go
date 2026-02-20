package presenters

import (
	"testing"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func TestAsJobSchedule(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	schedule := &core.Schedule{
		GUID:           "sched-guid-1",
		Enabled:        true,
		Expression:     "0 12 * * MON",
		ExpressionType: "cron",
		CreatedAt:      now,
		UpdatedAt:      now.Add(time.Hour),
		RefGUID:        "job-guid-1",
		RefType:        "job",
	}

	result := AsJobSchedule(schedule)

	if result.GUID != schedule.GUID {
		t.Errorf("GUID: got %q, want %q", result.GUID, schedule.GUID)
	}
	if result.Enabled != schedule.Enabled {
		t.Errorf("Enabled: got %v, want %v", result.Enabled, schedule.Enabled)
	}
	if result.Expression != schedule.Expression {
		t.Errorf("Expression: got %q, want %q", result.Expression, schedule.Expression)
	}
	if result.ExpressionType != schedule.ExpressionType {
		t.Errorf("ExpressionType: got %q, want %q", result.ExpressionType, schedule.ExpressionType)
	}
	if !result.CreatedAt.Equal(schedule.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", result.CreatedAt, schedule.CreatedAt)
	}
	if !result.UpdatedAt.Equal(schedule.UpdatedAt) {
		t.Errorf("UpdatedAt: got %v, want %v", result.UpdatedAt, schedule.UpdatedAt)
	}
	if result.JobGUID != schedule.RefGUID {
		t.Errorf("JobGUID: got %q, want %q", result.JobGUID, schedule.RefGUID)
	}
}

func TestAsJobScheduleCollection(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	schedules := []*core.Schedule{
		{GUID: "s1", Enabled: true, Expression: "*/10 * * * *", ExpressionType: "cron", CreatedAt: now, UpdatedAt: now, RefGUID: "j1"},
		{GUID: "s2", Enabled: false, Expression: "0 0 * * *", ExpressionType: "cron", CreatedAt: now, UpdatedAt: now, RefGUID: "j2"},
		{GUID: "s3", Enabled: true, Expression: "0 6 * * MON-FRI", ExpressionType: "cron", CreatedAt: now, UpdatedAt: now, RefGUID: "j3"},
	}

	result := AsJobScheduleCollection(schedules)

	if len(result) != 3 {
		t.Fatalf("length: got %d, want 3", len(result))
	}
	for i, sched := range schedules {
		if result[i].GUID != sched.GUID {
			t.Errorf("result[%d].GUID: got %q, want %q", i, result[i].GUID, sched.GUID)
		}
		if result[i].JobGUID != sched.RefGUID {
			t.Errorf("result[%d].JobGUID: got %q, want %q", i, result[i].JobGUID, sched.RefGUID)
		}
	}
}

func TestAsJobScheduleCollectionEmpty(t *testing.T) {
	result := AsJobScheduleCollection(nil)
	if len(result) != 0 {
		t.Errorf("length: got %d, want 0", len(result))
	}
}
