package logger

import (
	"testing"
	"time"
)

func TestTimestamp(t *testing.T) {
	var tests = []struct {
		name   string
		dur    time.Duration
		expect string
	}{
		{name: "zero", dur: 0, expect: "00h00m00s"},
		{name: "seconds only", dur: 45 * time.Second, expect: "00h00m45s"},
		{name: "minutes and seconds", dur: 3*time.Minute + 15*time.Second, expect: "00h03m15s"},
		{name: "hours minutes seconds", dur: 2*time.Hour + 30*time.Minute + 5*time.Second, expect: "02h30m05s"},
		{name: "large hours", dur: 100*time.Hour + 1*time.Minute + 1*time.Second, expect: "100h01m01s"},
		{name: "sub-second truncated", dur: 1*time.Second + 500*time.Millisecond, expect: "00h00m01s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timestamp(tt.dur)
			if result != tt.expect {
				t.Errorf("got %q, want %q", result, tt.expect)
			}
		})
	}
}
