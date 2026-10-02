package spec

import (
	"strings"
	"testing"
)

func TestFields(t *testing.T) {
	tests := []struct {
		expr  string
		names string
		loc   string
		desc  string
	}{
		{"*/15 * * * *", "minute hour day-of-month month day-of-week", "", ""},
		{"0 30 2 * * MON-FRI", "second minute hour day-of-month month day-of-week", "", ""},
		{"0 9 * * * 2027", "minute hour day-of-month month day-of-week year", "", ""},
		{"0 0 0 1 1 * 2027", "second minute hour day-of-month month day-of-week year", "", ""},
		{"CRON_TZ=America/Los_Angeles 0 2 * * *", "minute hour day-of-month month day-of-week", "America/Los_Angeles", ""},
		{`TZ="Europe/Berlin" @daily`, "", "Europe/Berlin", "@daily"},
		{"@every 1h30m", "", "", "@every"},
	}
	for _, tt := range tests {
		s, err := Fields(tt.expr)
		if err != nil {
			t.Fatalf("%q: %v", tt.expr, err)
		}
		var names []string
		for _, f := range s.Fields {
			names = append(names, f.Name)
			if tt.expr[f.Offset:f.Offset+len(f.Text)] != f.Text {
				t.Errorf("%q: field %s offset %d does not point at %q", tt.expr, f.Name, f.Offset, f.Text)
			}
		}
		if got := strings.Join(names, " "); got != tt.names || s.Location != tt.loc || s.Descriptor != tt.desc {
			t.Errorf("%q: got names %q loc %q desc %q", tt.expr, got, s.Location, s.Descriptor)
		}
	}
	for _, bad := range []string{"", "* * * *", "* * * * * * * *", "CRON_TZ=UTC"} {
		if _, err := Fields(bad); err == nil {
			t.Errorf("%q: want an error", bad)
		}
	}
}
