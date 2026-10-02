package cron

import (
	"strings"
	"testing"
	"time"

	cron "github.com/netresearch/go-cron"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// a fixed Monday, so results do not depend on when the tests run
var now = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

func codes(fs []core.Finding) string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Code)
	}
	return strings.Join(out, ",")
}

func TestAnalyzeErrors(t *testing.T) {
	tests := []struct {
		expr  string
		rules Rules
		want  string
	}{
		{"60 * * * *", Rules{}, "parse_error"},
		{"0 0 30 2 *", Rules{}, "never_fires"},
		{"0 0 1 1 * 2020", Rules{}, "never_fires"},
		{"@triggered", Rules{}, "never_fires"},
		{"*/30 * * * * *", Rules{MinInterval: time.Minute}, "under_min_interval"},
		{"@every 30s", Rules{MinInterval: time.Minute}, "under_min_interval"},
		{"*/30 * * * * *", Rules{}, ""},
		{"0 * * * *", Rules{MinInterval: time.Hour}, ""},
	}
	for _, tt := range tests {
		r := Analyze(tt.expr, "", 5, 0, tt.rules, now).Result
		if got := codes(r.Errors); got != tt.want {
			t.Errorf("%q: errors %q, want %q", tt.expr, got, tt.want)
		}
		if r.Valid != (tt.want == "") {
			t.Errorf("%q: valid %v", tt.expr, r.Valid)
		}
	}
}

func TestAnalyzeParseErrorIsStructured(t *testing.T) {
	r := Analyze("* 25 * * *", "", 5, 0, Rules{}, now).Result
	if len(r.Errors) == 0 {
		t.Fatalf("no errors found")
	}
	f := r.Errors[0]
	if f.Field != "hour" || f.Value != "25" || f.ValidRange != "0-23" || f.Offset == nil || *f.Offset != 2 {
		t.Errorf("got %+v", f)
	}
	if len(r.NextRuns) != 0 || r.Description == "" {
		t.Errorf("invalid: runs %v, description %q", r.NextRuns, r.Description)
	}
}

func TestAnalyzeWarnings(t *testing.T) {
	tests := []struct {
		expr, want string
	}{
		{"0 0 13 * FRI", "dom_and_dow"},
		{"0 0 31 * *", "dom_skips_months"},
		{"0 0 30,31 * *", "dom_skips_months"},
		{"0 0 31 1 *", ""},
		{"0 0 L * *", ""},
		{"CRON_TZ=UTC 0 1,,2 * * *", "empty_list_element"},
		{"CRON_TZ=America/Los_Angeles 30 2 * * *", "dst_skipped"},
		{"CRON_TZ=America/Los_Angeles 30 1 * * *", "dst_repeated"},
		{"CRON_TZ=America/Los_Angeles */5 * * * *", ""},
		{"CRON_TZ=UTC 30 2 * * *", ""},
	}
	for _, tt := range tests {
		r := Analyze(tt.expr, "", 5, 0, Rules{}, now).Result
		if got := codes(r.Warnings); got != tt.want {
			t.Errorf("%q: warnings %q, want %q (%v)", tt.expr, got, tt.want, r.Warnings)
		}
	}
}

func TestAnalyzeDSTMessages(t *testing.T) {
	r := Analyze("CRON_TZ=America/Los_Angeles 30 2 * * *", "", 5, 0, Rules{}, now).Result
	if len(r.Warnings) == 0 {
		t.Fatalf("no warnings found")
	}
	want := "02:30 on 2027-03-14 does not exist in America/Los_Angeles; it runs at 03:30"
	if r.Warnings[0].Message != want {
		t.Errorf("got %q", r.Warnings[0].Message)
	}
	r = Analyze("CRON_TZ=America/Los_Angeles 30 1 * * *", "", 5, 0, Rules{}, now).Result
	if len(r.Warnings) == 0 {
		t.Fatalf("no warnings found")
	}
	want = "01:30 on 2026-11-01 happens twice in America/Los_Angeles; it runs once"
	if r.Warnings[0].Message != want {
		t.Errorf("got %q", r.Warnings[0].Message)
	}
}

func TestAnalyzeHash(t *testing.T) {
	a := Analyze("H(15-45) 2 * * *", "job-guid-A", 3, 0, Rules{}, now)
	r := a.Result
	if !r.Illustrative || len(r.HashedFields) != 1 {
		t.Fatalf("got %+v", r)
	}
	h := r.HashedFields[0]
	if h.Field != "minute" || h.Value != "H(15-45)" || h.Resolved != "18" {
		t.Errorf("got %+v", h)
	}
	if len(r.NextRuns) == 0 {
		t.Fatalf("no next runs")
	}
	if r.NextRuns[0].Minute() != 18 {
		t.Errorf("next run %v does not use the resolved minute", r.NextRuns[0])
	}
	b := Analyze("H(15-45) 2 * * *", "job-guid-B", 3, 0, Rules{}, now).Result
	if len(b.HashedFields) != 1 {
		t.Fatalf("different key: hashed fields %+v", b.HashedFields)
	}
	if b.HashedFields[0].Resolved == "18" {
		t.Errorf("a different key resolved to the same minute")
	}
	if r := Analyze("H 2 * * *", "", 1, 0, Rules{}, now).Result; !r.Valid || !r.Illustrative {
		t.Errorf("placeholder key: %+v", r)
	}
	if r := Analyze("0 0 * * THU", "", 1, 0, Rules{}, now).Result; r.Illustrative {
		t.Errorf("THU is not a hash")
	}
}

func TestAnalyzeDescriptionNote(t *testing.T) {
	r := Analyze("H/15 * * * *", "", 1, 0, Rules{}, now).Result
	if r.Description != "every 15 minutes from a hashed start" ||
		r.DescriptionNote != "at minutes h, h+15, h+30 and h+45 of every hour, where h is a hashed minute from 0 to 14" {
		t.Errorf("got %q / %q", r.Description, r.DescriptionNote)
	}
	if r := Analyze("*/15 * * * *", "", 1, 0, Rules{}, now).Result; r.DescriptionNote != "" {
		t.Errorf("no hashed step, got note %q", r.DescriptionNote)
	}
}

func TestAnalyzeRuns(t *testing.T) {
	r := Analyze("CRON_TZ=UTC 0 9 * * *", "", 5, 2, Rules{}, now).Result
	if len(r.NextRuns) != 5 || len(r.PrevRuns) != 2 {
		t.Fatalf("next %d prev %d", len(r.NextRuns), len(r.PrevRuns))
	}
	if !r.NextRuns[0].Equal(time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)) ||
		!r.PrevRuns[0].Equal(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)) || r.Location != "UTC" {
		t.Errorf("next %v prev %v location %q", r.NextRuns[0], r.PrevRuns[0], r.Location)
	}
	if r := Analyze("* * * * *", "", 500, 500, Rules{}, now).Result; len(r.NextRuns) != MaxRuns || len(r.PrevRuns) != MaxRuns {
		t.Errorf("caps: next %d prev %d", len(r.NextRuns), len(r.PrevRuns))
	}
}

// Validate ≡ create: the Schedule handed to the cron service is the one the
// runs were computed from.
func TestAnalyzeScheduleMatchesRuns(t *testing.T) {
	for _, expr := range []string{"H(15-45) 2 * * MON-FRI", "*/7 * * * *", "CRON_TZ=Asia/Tokyo 0 9 * * *", "@every 90m"} {
		a := Analyze(expr, "job-guid-A", 5, 0, Rules{}, now)
		got := cron.NextN(a.Schedule, now, 5)
		if len(a.Result.NextRuns) != len(got) {
			t.Fatalf("%q: %d next runs, want %d", expr, len(a.Result.NextRuns), len(got))
		}
		for i := range got {
			if !got[i].Equal(a.Result.NextRuns[i]) {
				t.Errorf("%q: run %d %v != %v", expr, i, got[i], a.Result.NextRuns[i])
			}
		}
	}
}

// Inputs the design does not name (plan Review Focus).
func TestReviewFocusEdges(t *testing.T) {
	for _, tt := range []struct{ expr, desc, warn string }{
		{"  CRON_TZ=UTC 0 2 * * *  ", "at 02:00, UTC time", ""},
		{"CRON_TZ=UTC 0 0 * * mon-fri", "at 00:00, on Monday to Friday, UTC time", ""},
		{"CRON_TZ=UTC 0 0 ? * MON", "at 00:00, on Monday, UTC time", ""},
		{`CRON_TZ="America/New_York" 0 9 * * *`, "at 09:00, America/New_York time", ""},
		{"CRON_TZ=Australia/Lord_Howe 15 2 * * *", "at 02:15, Australia/Lord_Howe time", "dst_skipped"},
		{"CRON_TZ=America/Santiago 30 0 * * *", "at 00:30, America/Santiago time", "dst_skipped"},
	} {
		r := Analyze(tt.expr, "", 3, 0, Rules{}, now).Result
		if !r.Valid || r.Description != tt.desc || codes(r.Warnings) != tt.warn {
			t.Errorf("%q: valid %v desc %q warnings %v", tt.expr, r.Valid, r.Description, r.Warnings)
		}
	}
	for _, tt := range []struct{ expr, msg string }{
		{"CRON_TZ=Australia/Lord_Howe 15 2 * * *", "02:15 on 2027-10-03 does not exist in Australia/Lord_Howe; that run is skipped"},
		{"CRON_TZ=America/Santiago 30 0 * * *", "00:30 on 2027-09-05 does not exist in America/Santiago; that run is skipped"},
	} {
		if r := Analyze(tt.expr, "", 3, 0, Rules{}, now).Result; len(r.Warnings) == 0 {
			t.Fatalf("%q: no warnings", tt.expr)
		} else if r.Warnings[0].Message != tt.msg {
			t.Errorf("%q: %q", tt.expr, r.Warnings[0].Message)
		}
	}
}

// Matches go-cron's runner (cron.go isDSTFallBackDuplicate): a run is dropped
// only when it repeats the wall time of the last run kept.
func TestFallbackDuplicates(t *testing.T) {
	nextNow := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

	// next 3: Nov 1 08:30Z, Nov 2 09:30Z, Nov 3 09:30Z
	r := Analyze("CRON_TZ=America/Los_Angeles 30 1 * * *", "", 3, 0, Rules{}, nextNow).Result
	if len(r.NextRuns) != 3 {
		t.Fatalf("next: got %d runs, want 3", len(r.NextRuns))
	}
	expected := []time.Time{
		time.Date(2026, 11, 1, 8, 30, 0, 0, time.UTC),
		time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC),
		time.Date(2026, 11, 3, 9, 30, 0, 0, time.UTC),
	}
	for i, exp := range expected {
		if !r.NextRuns[i].Equal(exp) {
			t.Errorf("next[%d]: got %v, want %v", i, r.NextRuns[i], exp)
		}
	}

	// prev 2 from Nov 2: Nov 2 09:30Z, Nov 1 08:30Z
	prevNow := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	r = Analyze("CRON_TZ=America/Los_Angeles 30 1 * * *", "", 0, 2, Rules{}, prevNow).Result
	if len(r.PrevRuns) != 2 {
		t.Fatalf("prev: got %d runs, want 2", len(r.PrevRuns))
	}
	expectedPrev := []time.Time{
		time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC),
		time.Date(2026, 11, 1, 8, 30, 0, 0, time.UTC),
	}
	for i, exp := range expectedPrev {
		if !r.PrevRuns[i].Equal(exp) {
			t.Errorf("prev[%d]: got %v, want %v", i, r.PrevRuns[i], exp)
		}
	}

	// next 5: Nov 1 08:00Z, 08:30Z, 09:00Z, 09:30Z, Nov 2 09:00Z
	r = Analyze("CRON_TZ=America/Los_Angeles 0,30 1 * * *", "", 5, 0, Rules{}, nextNow).Result
	if len(r.NextRuns) != 5 {
		t.Fatalf("multi: got %d runs, want 5", len(r.NextRuns))
	}
	expectedMulti := []time.Time{
		time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 1, 8, 30, 0, 0, time.UTC),
		time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 1, 9, 30, 0, 0, time.UTC),
		time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC),
	}
	for i, exp := range expectedMulti {
		if !r.NextRuns[i].Equal(exp) {
			t.Errorf("multi[%d]: got %v, want %v", i, r.NextRuns[i], exp)
		}
	}
}

// dst_skipped says "it runs at HH:MM" only when the run is moved there and
// that time is not itself a regular run.
func TestDSTSkippedMessages(t *testing.T) {
	for _, tt := range []struct{ expr, msg string }{
		{"CRON_TZ=America/Los_Angeles 30 2 * * *", "02:30 on 2027-03-14 does not exist in America/Los_Angeles; it runs at 03:30"},
		{"CRON_TZ=America/Los_Angeles 30 2,3 * * *", "02:30 on 2027-03-14 does not exist in America/Los_Angeles; that run is skipped"},
		{"CRON_TZ=Australia/Lord_Howe 15 2,3 * * *", "02:15 on 2027-10-03 does not exist in Australia/Lord_Howe; that run is skipped"},
		{"CRON_TZ=Australia/Lord_Howe 15,45 2 * * *", "02:15 on 2027-10-03 does not exist in Australia/Lord_Howe; that run is skipped"},
		{"CRON_TZ=America/Santiago 30 0,6 * * *", "00:30 on 2027-09-05 does not exist in America/Santiago; that run is skipped"},
		{"CRON_TZ=America/Santiago 30 0,1 * * *", "00:30 on 2027-09-05 does not exist in America/Santiago; that run is skipped"},
	} {
		r := Analyze(tt.expr, "", 3, 0, Rules{}, now).Result
		if len(r.Warnings) == 0 {
			t.Errorf("%q: no warnings", tt.expr)
		} else if r.Warnings[0].Message != tt.msg {
			t.Errorf("%q: got %q, want %q", tt.expr, r.Warnings[0].Message, tt.msg)
		}
	}
}

// dst_repeated says how many times the runner fires the repeated wall time.
func TestDSTRepeatedMessages(t *testing.T) {
	for _, tt := range []struct{ expr, msg string }{
		{"CRON_TZ=America/Los_Angeles 30 1 * * *", "01:30 on 2026-11-01 happens twice in America/Los_Angeles; it runs once"},
		{"CRON_TZ=America/Los_Angeles 0,30 1 * * *", "01:00 on 2026-11-01 happens twice in America/Los_Angeles; it runs twice"},
	} {
		r := Analyze(tt.expr, "", 3, 0, Rules{}, now).Result
		if len(r.Warnings) == 0 {
			t.Errorf("%q: no warnings", tt.expr)
		} else if r.Warnings[0].Message != tt.msg {
			t.Errorf("%q: got %q, want %q", tt.expr, r.Warnings[0].Message, tt.msg)
		}
	}
}

// Without CRON_TZ the zone comes from now, not the process-local zone.
func TestZoneFromNow(t *testing.T) {
	// Temporarily set time.Local to Africa/Cairo
	oldLocal := time.Local
	loc, err := time.LoadLocation("Africa/Cairo")
	if err != nil {
		t.Skipf("Africa/Cairo not available: %v", err)
	}
	time.Local = loc
	defer func() { time.Local = oldLocal }()

	// now is UTC; Analyze should use now.Location() (UTC) for zone, not process's time.Local (Cairo)
	r := Analyze("0 0 30,31 * *", "", 3, 0, Rules{}, now).Result
	if r.Location != "UTC" {
		t.Errorf("location: got %q, want UTC", r.Location)
	}
	// Should only warn dom_skips_months, no dst_* warnings
	warn := codes(r.Warnings)
	if warn != "dom_skips_months" {
		t.Errorf("warnings: got %q, want dom_skips_months", warn)
	}
}

// dom_skips_months includes day 29 (February outside leap years).
func TestDomSkipsMonthsDay29(t *testing.T) {
	// 0 0 29 * * should warn dom_skips_months
	r := Analyze("0 0 29 * *", "", 3, 0, Rules{}, now).Result
	warn := codes(r.Warnings)
	if warn != "dom_skips_months" {
		t.Errorf("0 0 29 * *: warnings %q, want dom_skips_months", warn)
	}
	if len(r.Warnings) != 1 {
		t.Fatalf("0 0 29 * *: warnings %v", r.Warnings)
	}
	if want := "day 29 does not exist in February outside leap years; it does not run in those months"; r.Warnings[0].Message != want {
		t.Errorf("0 0 29 * *: got %q", r.Warnings[0].Message)
	}
	// 0 0 29 2 * should not warn dom_skips_months (only February selected)
	r = Analyze("0 0 29 2 *", "", 3, 0, Rules{}, now).Result
	warn = codes(r.Warnings)
	if warn != "" {
		t.Errorf("0 0 29 2 *: warnings %q, want none", warn)
	}
}

func checkRuns(t *testing.T, name string, got []time.Time, want []time.Time) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %d runs, want %d: %v", name, len(got), len(want), got)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("%s[%d]: got %v, want %v", name, i, got[i].UTC(), want[i])
		}
	}
}

// 5-minute runs in the repeated hour: 08:00Z..09:55Z, none dropped, and the
// window size must not change which instants appear.
func TestFallbackFiveMinuteRuns(t *testing.T) {
	const expr = "CRON_TZ=America/Los_Angeles */5 1 * * *"
	nextNow := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	r := Analyze(expr, "", 30, 0, Rules{}, nextNow).Result
	if len(r.NextRuns) != 30 {
		t.Fatalf("next: %d runs", len(r.NextRuns))
	}
	for i := 0; i < 24; i++ {
		want := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC).Add(time.Duration(i) * 5 * time.Minute)
		if !r.NextRuns[i].Equal(want) {
			t.Errorf("next[%d]: got %v, want %v", i, r.NextRuns[i].UTC(), want)
		}
	}
	if want := time.Date(2026, 11, 2, 9, 25, 0, 0, time.UTC); !r.NextRuns[29].Equal(want) {
		t.Errorf("next[29]: got %v, want %v", r.NextRuns[29].UTC(), want)
	}

	prevNow := time.Date(2026, 11, 1, 11, 0, 0, 0, time.UTC)
	r = Analyze(expr, "", 0, 3, Rules{}, prevNow).Result
	checkRuns(t, "prev3", r.PrevRuns, []time.Time{
		time.Date(2026, 11, 1, 9, 55, 0, 0, time.UTC),
		time.Date(2026, 11, 1, 9, 50, 0, 0, time.UTC),
		time.Date(2026, 11, 1, 9, 45, 0, 0, time.UTC),
	})
	r = Analyze(expr, "", 0, 24, Rules{}, prevNow).Result
	var all []time.Time
	for i := 0; i < 24; i++ {
		all = append(all, time.Date(2026, 11, 1, 9, 55, 0, 0, time.UTC).Add(-time.Duration(i)*5*time.Minute))
	}
	checkRuns(t, "prev24", r.PrevRuns, all)
}

// The 09:30Z run is the skipped duplicate, so it is not a previous run.
func TestFallbackPrevSkipsDuplicate(t *testing.T) {
	prevNow := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	r := Analyze("CRON_TZ=America/Los_Angeles 30 1 * * *", "", 0, 1, Rules{}, prevNow).Result
	checkRuns(t, "prev1", r.PrevRuns, []time.Time{time.Date(2026, 11, 1, 8, 30, 0, 0, time.UTC)})
}

// A repeated wall time that the runner drops is not a second run, so it must
// not count toward the minimum interval.
func TestMinIntervalIgnoresFallbackRepeats(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"CRON_TZ=America/Los_Angeles 0 1 * * *", ""},
		{"CRON_TZ=America/Los_Angeles 30 1 * * SUN", ""},
		{"CRON_TZ=Europe/Berlin 30 2 * * *", ""},
		{"CRON_TZ=America/Los_Angeles */30 * * * *", "under_min_interval"},
	}
	for _, tt := range tests {
		r := Analyze(tt.expr, "", 5, 0, Rules{MinInterval: 2 * time.Hour}, now).Result
		if got := codes(r.Errors); got != tt.want {
			t.Errorf("%q: errors %q (%v), want %q", tt.expr, got, r.Errors, tt.want)
		}
	}
}
func TestNeverFiresSearchLimit(t *testing.T) {
	r := Analyze("CRON_TZ=UTC 0 0 13 2 FRI", "", 5, 0, Rules{}, now).Result
	want := "next run is on 2032-02-13, beyond the scheduler's 5-year search limit; it would not be scheduled"
	if len(r.Errors) != 1 || r.Errors[0].Code != "never_fires" || r.Errors[0].Message != want {
		t.Errorf("rare date: %+v, want never_fires %q", r.Errors, want)
	}
	r = Analyze("CRON_TZ=UTC 0 0 30 2 *", "", 5, 0, Rules{}, now).Result
	if len(r.Errors) != 1 || r.Errors[0].Code != "never_fires" || r.Errors[0].Message != "this schedule never runs" {
		t.Errorf("impossible date: %+v", r.Errors)
	}
}
