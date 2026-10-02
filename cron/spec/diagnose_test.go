package spec

import (
	"testing"

	cron "github.com/netresearch/go-cron"
)

func TestDiagnose(t *testing.T) {
	tests := []struct {
		expr                     string
		field, value, valid, msg string
		offset                   int
	}{
		{"60 * * * *", "minute", "60", "0-59", "minute field value 60 is out of range (valid: 0-59)", 0},
		{"* 25 * * *", "hour", "25", "0-23", "hour field value 25 is out of range (valid: 0-23)", 2},
		{"* * 32 * *", "day-of-month", "32", "1-31", "", 4},
		{"* * * 13 *", "month", "13", "1-12", "", 6},
		{"* * * * 8", "day-of-week", "8", "0-7", "", 8},
		{"*/0 * * * *", "minute", "*/0", "", "minute field value */0 has a step below 1", 0},
		{"* * * * MON-FUN", "day-of-week", "FUN", "0-7", "day-of-week field value FUN is not a number or name (valid: 0-7)", 12},
		{"0 0 L-31 * *", "day-of-month", "L-31", "", "", 4},
		{"0 0 * * MON#6", "day-of-week", "MON#6", "", "", 8},
		{"5,70 * * * *", "minute", "70", "0-59", "", 2},
		{"H(0-60) * * * *", "minute", "60", "0-59", "", 4},
		{"CRON_TZ=Mars/Base * * * * *", "time zone", "Mars/Base", "", "time zone field value Mars/Base is not a known time zone", 8},
		{"@fortnightly", "descriptor", "@fortnightly", descriptors, "", 0},
		{"@every 0s", "@every", "0s", "", "", 7},
		{"* * * *", "", "", "", "expected 5 to 7 fields, found 4", -1},
	}
	for _, tt := range tests {
		_, perr := cron.FullParser().WithHashKey("k").Parse(tt.expr)
		if perr == nil {
			t.Errorf("%q: go-cron accepted it", tt.expr)
			continue
		}
		e := Diagnose(tt.expr, perr)
		if e.Field != tt.field || e.Value != tt.value || e.ValidRange != tt.valid || e.Offset != tt.offset {
			t.Errorf("%q: got field %q value %q valid %q offset %d", tt.expr, e.Field, e.Value, e.ValidRange, e.Offset)
		}
		if tt.msg != "" && e.Message() != tt.msg {
			t.Errorf("%q: message %q, want %q", tt.expr, e.Message(), tt.msg)
		}
	}
}

func TestDiagnoseFallsBackToGoCron(t *testing.T) {
	err := fmtErr("something go-cron says")
	e := Diagnose("0 0 * * *", err)
	if e.Field != "" || e.Reason != "something go-cron says" || e.Offset != -1 {
		t.Errorf("got %+v", e)
	}
}

type fmtErr string

func (e fmtErr) Error() string { return string(e) }
