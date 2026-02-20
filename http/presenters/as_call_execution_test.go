package presenters

import (
	"testing"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func TestAsCallExecution(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	execution := &core.Execution{
		GUID:               "exec-guid-1",
		RefGUID:            "call-guid-1",
		ScheduleGUID:       "sched-guid-1",
		ScheduledTime:      now,
		Message:            "completed successfully",
		State:              "SUCCEEDED",
		ExecutionStartTime: now.Add(-time.Minute),
		ExecutionEndTime:   now,
		TaskGUID:           "should-not-appear",
		RefType:            "call",
	}

	result := AsCallExecution(execution)

	if result.GUID != execution.GUID {
		t.Errorf("GUID: got %q, want %q", result.GUID, execution.GUID)
	}
	if result.CallGUID != execution.RefGUID {
		t.Errorf("CallGUID: got %q, want %q", result.CallGUID, execution.RefGUID)
	}
	if result.ScheduleGUID != execution.ScheduleGUID {
		t.Errorf("ScheduleGUID: got %q, want %q", result.ScheduleGUID, execution.ScheduleGUID)
	}
	if !result.ScheduledTime.Equal(execution.ScheduledTime) {
		t.Errorf("ScheduledTime: got %v, want %v", result.ScheduledTime, execution.ScheduledTime)
	}
	if result.Message != execution.Message {
		t.Errorf("Message: got %q, want %q", result.Message, execution.Message)
	}
	if result.State != execution.State {
		t.Errorf("State: got %q, want %q", result.State, execution.State)
	}
	if !result.ExecutionStartTime.Equal(execution.ExecutionStartTime) {
		t.Errorf("ExecutionStartTime: got %v, want %v", result.ExecutionStartTime, execution.ExecutionStartTime)
	}
	if !result.ExecutionEndTime.Equal(execution.ExecutionEndTime) {
		t.Errorf("ExecutionEndTime: got %v, want %v", result.ExecutionEndTime, execution.ExecutionEndTime)
	}
}

func TestAsCallExecutionCollection(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	executions := []*core.Execution{
		{GUID: "e1", RefGUID: "c1", State: "SUCCEEDED", ScheduledTime: now, ExecutionStartTime: now, ExecutionEndTime: now},
		{GUID: "e2", RefGUID: "c2", State: "FAILED", ScheduledTime: now, ExecutionStartTime: now, ExecutionEndTime: now},
	}

	result := AsCallExecutionCollection(executions)

	if len(result) != 2 {
		t.Fatalf("length: got %d, want 2", len(result))
	}
	if result[0].GUID != "e1" {
		t.Errorf("result[0].GUID: got %q, want %q", result[0].GUID, "e1")
	}
	if result[1].GUID != "e2" {
		t.Errorf("result[1].GUID: got %q, want %q", result[1].GUID, "e2")
	}
	if result[0].CallGUID != "c1" {
		t.Errorf("result[0].CallGUID: got %q, want %q", result[0].CallGUID, "c1")
	}
}

func TestAsCallExecutionCollectionEmpty(t *testing.T) {
	result := AsCallExecutionCollection(nil)
	if len(result) != 0 {
		t.Errorf("length: got %d, want 0", len(result))
	}
}
