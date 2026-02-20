package core

import (
	"strings"
	"testing"
)

func TestGenGUID(t *testing.T) {
	guid, err := GenGUID()
	if err != nil {
		t.Fatalf("GenGUID() error: %v", err)
	}

	if len(guid) == 0 {
		t.Fatal("GenGUID() returned empty string")
	}

	// UUID v4 format: 8-4-4-4-12 hex chars
	parts := strings.Split(guid, "-")
	if len(parts) != 5 {
		t.Errorf("expected 5 UUID parts, got %d: %q", len(parts), guid)
	}

	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			t.Errorf("part %d: got length %d, want %d", i, len(part), expectedLengths[i])
		}
	}
}

func TestGenGUIDUniqueness(t *testing.T) {
	guid1, err := GenGUID()
	if err != nil {
		t.Fatalf("GenGUID() first call error: %v", err)
	}

	guid2, err := GenGUID()
	if err != nil {
		t.Fatalf("GenGUID() second call error: %v", err)
	}

	if guid1 == guid2 {
		t.Errorf("two calls to GenGUID() returned the same value: %q", guid1)
	}
}
