package presenters

import (
	"testing"
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func TestAsJobExecution(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	execution := &core.Execution{
		GUID:               "exec-guid-1",
		RefGUID:            "job-guid-1",
		TaskGUID:           "task-guid-1",
		ScheduleGUID:       "sched-guid-1",
		ScheduledTime:      now,
		Message:            "task completed",
		State:              "SUCCEEDED",
		ExecutionStartTime: now.Add(-2 * time.Minute),
		ExecutionEndTime:   now,
		RefType:            "job",
	}

	result := AsJobExecution(execution)

	if result.GUID != execution.GUID {
		t.Errorf("GUID: got %q, want %q", result.GUID, execution.GUID)
	}
	if result.JobGUID != execution.RefGUID {
		t.Errorf("JobGUID: got %q, want %q", result.JobGUID, execution.RefGUID)
	}
	if result.TaskGUID != execution.TaskGUID {
		t.Errorf("TaskGUID: got %q, want %q", result.TaskGUID, execution.TaskGUID)
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

func TestAsJobExecutionEmptyScheduleGUID(t *testing.T) {
	execution := &core.Execution{
		GUID:    "exec-guid-2",
		RefGUID: "job-guid-2",
		State:   "PENDING",
	}

	result := AsJobExecution(execution)

	if result.ScheduleGUID != "" {
		t.Errorf("ScheduleGUID: got %q, want empty", result.ScheduleGUID)
	}
}

func TestAsJobExecutionCollection(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	executions := []*core.Execution{
		{GUID: "e1", RefGUID: "j1", TaskGUID: "t1", State: "SUCCEEDED", ScheduledTime: now, ExecutionStartTime: now, ExecutionEndTime: now},
		{GUID: "e2", RefGUID: "j2", TaskGUID: "t2", State: "FAILED", ScheduledTime: now, ExecutionStartTime: now, ExecutionEndTime: now},
	}

	result := AsJobExecutionCollection(executions)

	if len(result) != 2 {
		t.Fatalf("length: got %d, want 2", len(result))
	}
	if result[0].JobGUID != "j1" {
		t.Errorf("result[0].JobGUID: got %q, want %q", result[0].JobGUID, "j1")
	}
	if result[1].TaskGUID != "t2" {
		t.Errorf("result[1].TaskGUID: got %q, want %q", result[1].TaskGUID, "t2")
	}
}

func TestAsJobExecutionCollectionEmpty(t *testing.T) {
	result := AsJobExecutionCollection(nil)
	if len(result) != 0 {
		t.Errorf("length: got %d, want 0", len(result))
	}
}
