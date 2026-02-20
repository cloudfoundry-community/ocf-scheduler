package presenters

import (
	"testing"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func TestAsCallSchedule(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	schedule := &core.Schedule{
		GUID:           "sched-guid-1",
		Enabled:        true,
		Expression:     "*/5 * * * *",
		ExpressionType: "cron",
		CreatedAt:      now,
		UpdatedAt:      now.Add(time.Hour),
		RefGUID:        "call-guid-1",
		RefType:        "call",
	}

	result := AsCallSchedule(schedule)

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
	if result.CallGUID != schedule.RefGUID {
		t.Errorf("CallGUID: got %q, want %q", result.CallGUID, schedule.RefGUID)
	}
}

func TestAsCallScheduleCollection(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	schedules := []*core.Schedule{
		{GUID: "s1", Enabled: true, Expression: "*/5 * * * *", ExpressionType: "cron", CreatedAt: now, UpdatedAt: now, RefGUID: "c1"},
		{GUID: "s2", Enabled: false, Expression: "0 0 * * *", ExpressionType: "cron", CreatedAt: now, UpdatedAt: now, RefGUID: "c2"},
	}

	result := AsCallScheduleCollection(schedules)

	if len(result) != 2 {
		t.Fatalf("length: got %d, want 2", len(result))
	}
	if result[0].GUID != "s1" {
		t.Errorf("result[0].GUID: got %q, want %q", result[0].GUID, "s1")
	}
	if result[1].GUID != "s2" {
		t.Errorf("result[1].GUID: got %q, want %q", result[1].GUID, "s2")
	}
	if result[0].CallGUID != "c1" {
		t.Errorf("result[0].CallGUID: got %q, want %q", result[0].CallGUID, "c1")
	}
	if result[1].CallGUID != "c2" {
		t.Errorf("result[1].CallGUID: got %q, want %q", result[1].CallGUID, "c2")
	}
}

func TestAsCallScheduleCollectionEmpty(t *testing.T) {
	result := AsCallScheduleCollection(nil)
	if len(result) != 0 {
		t.Errorf("length: got %d, want 0", len(result))
	}
}
