package core

import (
	"testing"
)

func TestCallType(t *testing.T) {
	call := &Call{GUID: "test-guid", Name: "my-call"}
	if got := call.Type(); got != "call" {
		t.Errorf("Type(): got %q, want %q", got, "call")
	}
}

func TestCallToCall(t *testing.T) {
	call := &Call{GUID: "test-guid", Name: "my-call", URL: "https://example.com"}
	result, err := call.ToCall()
	if err != nil {
		t.Fatalf("ToCall() error: %v", err)
	}
	if result != call {
		t.Errorf("ToCall() returned a different pointer")
	}
}

func TestCallToJob(t *testing.T) {
	call := &Call{GUID: "test-guid"}
	result, err := call.ToJob()
	if err == nil {
		t.Fatal("ToJob() expected error, got nil")
	}
	if result != nil {
		t.Errorf("ToJob() expected nil result, got %v", result)
	}
}
