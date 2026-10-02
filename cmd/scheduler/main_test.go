package main

import (
	"errors"
	"testing"
	"time"
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

func TestParseMinInterval(t *testing.T) {
	var tests = []struct {
		raw     string
		expect  time.Duration
		wantErr bool
	}{
		{raw: "", expect: 0},
		{raw: "0", expect: 0},
		{raw: "30s", expect: 30 * time.Second},
		{raw: "1h", expect: time.Hour},
		{raw: "soon", wantErr: true},
		{raw: "-1m", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := parseMinInterval(tt.raw)
			if (err != nil) != tt.wantErr || got != tt.expect {
				t.Errorf("got %v, %v; want %v, error %v", got, err, tt.expect, tt.wantErr)
			}
		})
	}
}
