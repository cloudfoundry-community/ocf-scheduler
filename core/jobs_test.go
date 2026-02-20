package core

import (
	"testing"
)

func TestJobType(t *testing.T) {
	job := &Job{GUID: "test-guid", Name: "my-job"}
	if got := job.Type(); got != "job" {
		t.Errorf("Type(): got %q, want %q", got, "job")
	}
}

func TestJobToJob(t *testing.T) {
	job := &Job{GUID: "test-guid", Name: "my-job", Command: "echo hello"}
	result, err := job.ToJob()
	if err != nil {
		t.Fatalf("ToJob() error: %v", err)
	}
	if result != job {
		t.Errorf("ToJob() returned a different pointer")
	}
}

func TestJobToCall(t *testing.T) {
	job := &Job{GUID: "test-guid"}
	result, err := job.ToCall()
	if err == nil {
		t.Fatal("ToCall() expected error, got nil")
	}
	if result != nil {
		t.Errorf("ToCall() expected nil result, got %v", result)
	}
}
