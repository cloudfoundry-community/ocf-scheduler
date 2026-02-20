package main

import (
	"errors"
	"testing"
)

func TestErrorString(t *testing.T) {
	var tests = []struct {
		name   string
		err    error
		expect string
	}{
		{name: "nil error", err: nil, expect: ""},
		{name: "non-nil error", err: errors.New("something failed"), expect: "something failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ErrorString(tt.err)
			if result != tt.expect {
				t.Errorf("got %q, want %q", result, tt.expect)
			}
		})
	}
}
