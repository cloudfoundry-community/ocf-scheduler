package core

import "testing"

func TestGenGUID(t *testing.T) {
    guid, err := GenGUID()
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if len(guid) == 0 { t.Fatalf("expected non-empty guid") }
}
