package cron

import (
	"fmt"
	"math/bits"
	"slices"
	"strconv"
	"strings"
	"time"

	cron "github.com/netresearch/go-cron"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/cron/spec"
)

// PlaceholderKey hashes H for an expression validated without a job or call.
const PlaceholderKey = "ocf-scheduler-validate"

// MaxRuns caps next and prev.
const MaxRuns = 100

// starBit mirrors go-cron's unexported marker for a "*" field (spec.go).
const starBit = 1 << 63

// Rules are the scheduler's own limits on top of what go-cron accepts.
type Rules struct {
	MinInterval time.Duration
}

// Analysis is the result of Analyze plus the parsed schedule for the cron
// service to register.
type Analysis struct {
	Result   core.ScheduleAnalysis
	Schedule cron.Schedule // nil when the expression does not parse
}

// Analyze parses expression exactly as the cron service will (H keyed by
// key, the job or call GUID) and reports errors, warnings, a description and
// the next and previous runs from now. It is the one place the scheduler
// decides what an expression means: validation and scheduling both use it.
func Analyze(expression, key string, next, prev int, rules Rules, now time.Time) Analysis {
	expression = strings.TrimSpace(expression)
	if key == "" {
		key = PlaceholderKey
	}
	r := core.ScheduleAnalysis{
		Expression:   expression,
		From:         now,
		HashedFields: []core.HashedField{},
		NextRuns:     []time.Time{},
		PrevRuns:     []time.Time{},
		Errors:       []core.Finding{},
		Warnings:     []core.Finding{},
	}
	sched, err := cron.FullParser().WithHashKey(key).Parse(expression)
	if err != nil {
		pe := spec.Diagnose(expression, err)
		f := core.Finding{Code: "parse_error", Message: pe.Message(), Field: pe.Field,
			Value: pe.Value, ValidRange: pe.ValidRange}
		if pe.Offset >= 0 {
			f.Offset = &pe.Offset
		}
		r.Errors = append(r.Errors, f)
		r.Description, _ = spec.Describe(expression)
		return Analysis{Result: r}
	}
	r.Description, _ = spec.Describe(expression)
	r.DescriptionNote = spec.DescribeNote(expression)
	fields, _ := spec.Fields(expression)
	ss, isSpec := sched.(*cron.SpecSchedule)

	// A schedule without CRON_TZ uses time.Local; report the zone of now instead.
	var effectiveZone *time.Location
	if isSpec {
		if ss.Location == time.Local {
			effectiveZone = now.Location()
		} else {
			effectiveZone = ss.Location
		}
		r.Location = effectiveZone.String()
		r.HashedFields = hashedFields(fields, ss)
		r.Illustrative = len(r.HashedFields) > 0
	}

	// Run lists follow go-cron's runner: a run is dropped only when it repeats
	// the wall time of the run kept just before it (see isDSTFallbackDuplicate).
	nextRuns := runnerRuns(sched, now, clamp(next), effectiveZone)

	prevRuns := []time.Time{}
	if isSpec && effectiveZone != nil {
		// PrevRuns: fetch extra, sort to chronological order, apply rule ascending, return newest first
		rawPrev := cron.PrevN(sched, now, clamp(prev)*2+1)
		if len(rawPrev) > 0 {
			// Sort to chronological (ascending) order for processing
			sortAsc := make([]time.Time, len(rawPrev))
			for i := range rawPrev {
				sortAsc[i] = rawPrev[len(rawPrev)-1-i]
			}
			// Apply the rule: keep run unless it's a duplicate of the last KEPT
			var lastKept time.Time
			kept := []time.Time{}
			for _, t := range sortAsc {
				if !isDSTFallbackDuplicate(lastKept, t, effectiveZone) {
					kept = append(kept, t)
					lastKept = t
				}
			}
			// Take the last clamp(prev) runs (newest), then reverse to return newest first
			if len(kept) > clamp(prev) {
				kept = kept[len(kept)-clamp(prev):]
			}
			prevRuns = kept
			for i := len(prevRuns)/2 - 1; i >= 0; i-- {
				opp := len(prevRuns) - 1 - i
				prevRuns[i], prevRuns[opp] = prevRuns[opp], prevRuns[i]
			}
		}
	} else {
		// No zone info, just use raw runs
		prevRuns = cron.PrevN(sched, now, clamp(prev))
	}

	r.NextRuns = orEmpty(nextRuns)
	r.PrevRuns = orEmpty(prevRuns)

	if sched.Next(now).IsZero() {
		r.Errors = append(r.Errors, core.Finding{Code: "never_fires",
			Message: neverFiresMessage(expression, key, now, effectiveZone)})
	} else if rules.MinInterval > 0 {
		if gap, ok := smallestGap(sched, now, effectiveZone); ok && gap < rules.MinInterval {
			r.Errors = append(r.Errors, core.Finding{Code: "under_min_interval",
				Message: fmt.Sprintf("runs as often as every %s; this scheduler's minimum is %s", gap, rules.MinInterval)})
		}
	}
	r.Valid = len(r.Errors) == 0

	r.Warnings = append(r.Warnings, emptyElements(fields)...)
	if isSpec {
		r.Warnings = append(r.Warnings, domWarnings(ss)...)
		if len(r.Errors) == 0 && effectiveZone != nil {
			r.Warnings = append(r.Warnings, dstWarnings(ss, now, nextRuns, effectiveZone)...)
		}
	}
	return Analysis{Result: r, Schedule: sched}
}

func clamp(n int) int { return max(0, min(n, MaxRuns)) }

func orEmpty(ts []time.Time) []time.Time {
	if ts == nil {
		return []time.Time{}
	}
	return ts
}

func fieldMask(ss *cron.SpecSchedule, name string) (uint64, bool) {
	switch name {
	case spec.Second:
		return ss.Second, true
	case spec.Minute:
		return ss.Minute, true
	case spec.Hour:
		return ss.Hour, true
	case spec.DayOfMonth:
		return ss.Dom, true
	case spec.Month:
		return ss.Month, true
	case spec.DayOfWeek:
		return ss.Dow, true
	}
	return 0, false
}

func setBits(mask uint64) []int {
	var out []int
	mask &^= starBit
	for mask != 0 {
		b := bits.TrailingZeros64(mask)
		out = append(out, b)
		mask &^= 1 << b
	}
	return out
}

func hashedFields(s spec.Spec, ss *cron.SpecSchedule) []core.HashedField {
	out := []core.HashedField{}
	for _, f := range s.Fields {
		hashed := false
		for _, e := range strings.Split(f.Text, ",") {
			if e == "H" || strings.HasPrefix(e, "H(") || strings.HasPrefix(e, "H/") {
				hashed = true
			}
		}
		mask, ok := fieldMask(ss, f.Name)
		if !hashed || !ok {
			continue
		}
		vals := setBits(mask)
		words := make([]string, len(vals))
		for i, v := range vals {
			words[i] = strconv.Itoa(v)
		}
		out = append(out, core.HashedField{Field: f.Name, Value: f.Text, Resolved: strings.Join(words, ",")})
	}
	return out
}

// runnerRuns returns the next n runs go-cron's runner would fire: it steps
// through Schedule.Next and drops a run only when it repeats the wall time of
// the run kept just before it (see isDSTFallbackDuplicate). A nil loc means no
// zone to judge repeats by, so the raw runs are returned.
func runnerRuns(sched cron.Schedule, now time.Time, n int, loc *time.Location) []time.Time {
	if loc == nil {
		return cron.NextN(sched, now, n)
	}
	runs := []time.Time{}
	var lastKept time.Time
	for t := now; len(runs) < n; {
		t = sched.Next(t)
		if t.IsZero() {
			break
		}
		if !isDSTFallbackDuplicate(lastKept, t, loc) {
			runs = append(runs, t)
			lastKept = t
		}
	}
	return runs
}

// neverFiresMessage explains a schedule go-cron's 5-year search found no run
// for. A rare date (Feb 29 on a Monday) can still have one further out; the
// scheduler cannot register it, so it stays an error, but the message says so.
func neverFiresMessage(expression, key string, now time.Time, loc *time.Location) string {
	const never = "this schedule never runs"
	sched, err := cron.FullParser().WithHashKey(key).WithMaxSearchYears(100).Parse(expression)
	if err != nil {
		return never
	}
	t := sched.Next(now)
	if t.IsZero() {
		return never
	}
	if loc == nil {
		loc = now.Location()
	}
	return fmt.Sprintf("next run is on %s, beyond the scheduler's 5-year search limit; it would not be scheduled",
		t.In(loc).Format("2006-01-02"))
}

// smallestGap is the smallest time between consecutive runs among the next
// 1000 the runner fires.
// NOTE: a sample, not a proof; fields are ANDed so the smallest gap shows up
// early. Compute it from the field bitmasks if that ever falls short.
func smallestGap(sched cron.Schedule, now time.Time, loc *time.Location) (time.Duration, bool) {
	if cd, ok := sched.(cron.ConstantDelaySchedule); ok {
		return cd.Delay, true
	}
	runs := runnerRuns(sched, now, 1000, loc)
	if len(runs) < 2 {
		return 0, false
	}
	gap := runs[1].Sub(runs[0])
	for i := 2; i < len(runs); i++ {
		gap = min(gap, runs[i].Sub(runs[i-1]))
	}
	return gap, true
}

func emptyElements(s spec.Spec) []core.Finding {
	var out []core.Finding
	for _, f := range s.Fields {
		for _, e := range strings.Split(f.Text, ",") {
			if e == "" {
				out = append(out, core.Finding{Code: "empty_list_element", Field: f.Name, Value: f.Text,
					Message: fmt.Sprintf("%s field %s has an empty list element; it is ignored", f.Name, f.Text)})
				break
			}
		}
	}
	return out
}

var monthNames = []string{"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}

var monthDays = []int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

func domWarnings(ss *cron.SpecSchedule) []core.Finding {
	var out []core.Finding
	domRestricted := ss.Dom&starBit == 0 || len(ss.DomConstraints) > 0
	dowRestricted := ss.Dow&starBit == 0 || len(ss.DowConstraints) > 0
	if domRestricted && dowRestricted {
		out = append(out, core.Finding{Code: "dom_and_dow",
			Message: "day-of-month and day-of-week are both restricted: it runs only on days that match both"})
	}
	if ss.Dom&starBit != 0 && len(ss.DomConstraints) == 0 {
		return out
	}
	var days []int
	if ss.Dom&starBit == 0 {
		days = setBits(ss.Dom)
	}
	for _, c := range ss.DomConstraints {
		if c.Type != cron.DomNearestWeekday {
			return out // L, LW and L-n fall in every month
		}
		days = append(days, c.N) // nW does not run where day n does not exist
	}
	slices.Sort(days)
	if len(days) == 0 || days[0] < 29 {
		return out
	}
	var short []string
	months := setBits(ss.Month)
	for _, m := range months {
		if m == 2 && days[0] == 29 {
			// day 29 exists in February in leap years only
			short = append(short, monthNames[m])
		} else if monthDays[m] < days[0] {
			short = append(short, monthNames[m])
		}
	}
	if len(short) > 0 && len(short) < len(months) {
		dayName := fmt.Sprintf("day %d", days[0])
		if days[0] == 29 {
			dayName = "day 29"
			// Check if February is in short
			for i, s := range short {
				if s == "February" {
					short[i] = "February outside leap years"
					break
				}
			}
		}
		out = append(out, core.Finding{Code: "dom_skips_months",
			Message: fmt.Sprintf("%s does not exist in %s; it does not run in those months", dayName, joinAnd(short))})
	}
	return out
}

func joinAnd(xs []string) string {
	if len(xs) == 1 {
		return xs[0]
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " and " + xs[len(xs)-1]
}

// dstWarnings reports runs that fall in a DST gap or overlap in the next 12
// months, or up to the last listed run when that is later, so a year-pinned
// expression warns the same whenever it is checked. Only for schedules that restrict the hour: an every-minute or
// hourly schedule losing or repeating one hour is expected.
//
// The intended wall-clock runs come from a copy of the schedule in UTC, which
// has no DST. What go-cron really does comes from the schedule itself,
// stepping from the previous run as the scheduler loop does: it moves some
// skipped runs (02:30 → 03:30 in America/Los_Angeles) and drops others
// (30-minute and midnight transitions), so the message says which.
func dstWarnings(ss *cron.SpecSchedule, now time.Time, listed []time.Time, loc *time.Location) []core.Finding {
	if ss.Hour&starBit != 0 || loc == nil {
		return nil
	}
	until := now.AddDate(1, 0, 0)
	if n := len(listed); n > 0 && listed[n-1].After(until) {
		until = listed[n-1].Add(time.Second)
	}
	naive := *ss
	naive.Location = time.UTC
	var out []core.Finding
	for _, tr := range transitions(loc, now, until) {
		if tr.at.After(until) {
			continue // the hourly scan looks up to an hour past until
		}
		delta := time.Duration(tr.after-tr.before) * time.Second
		gap := delta > 0
		offset := tr.after
		if gap {
			offset = tr.before
		}
		// the skipped (gap) or repeated (overlap) wall-clock interval, as UTC
		w := tr.at.In(time.FixedZone("", offset))
		start := time.Date(w.Year(), w.Month(), w.Day(), w.Hour(), w.Minute(), w.Second(), 0, time.UTC)
		intended := cron.BetweenWithLimit(&naive, start.Add(-time.Second), start.Add(delta.Abs()), 1)
		if len(intended) == 0 {
			continue
		}
		wall, date := intended[0].Format("15:04"), intended[0].Format("2006-01-02")
		if !gap {
			times := "once"
			if runnerFires(ss, tr.at.Add(-delta.Abs()), tr.at.Add(delta.Abs()), wall, loc) == 2 {
				times = "twice"
			}
			out = append(out, core.Finding{Code: "dst_repeated",
				Message: fmt.Sprintf("%s on %s happens twice in %s; it runs %s", wall, date, loc, times)})
			continue
		}
		// Claim "it runs at HH:MM" only when the run is moved to exactly that
		// time and HH:MM is not itself a regular run that day.
		what := "that run is skipped"
		moved := intended[0].Add(delta)
		regular := naive.Next(moved.Add(-time.Second)).Equal(moved)
		// step as the runner does: from the run before the gap, or from now
		// when there is none (the first run of a year-pinned schedule)
		from := now
		if prev := cron.PrevN(ss, tr.at, 1); len(prev) == 1 {
			from = prev[0]
		}
		if next := ss.Next(from).In(loc); !regular &&
			next.Format("2006-01-02") == date && next.Format("15:04") == moved.Format("15:04") {
			what = "it runs at " + next.Format("15:04")
		}
		out = append(out, core.Finding{Code: "dst_skipped",
			Message: fmt.Sprintf("%s on %s does not exist in %s; %s", wall, date, loc, what)})
	}
	return out
}

// runnerFires counts how often go-cron's runner fires the given wall time
// (HH:MM) among the runs in [from, to), dropping fall-back duplicates.
func runnerFires(sched cron.Schedule, from, to time.Time, wall string, loc *time.Location) int {
	var last time.Time
	n := 0
	for t := sched.Next(from.Add(-time.Second)); !t.IsZero() && t.Before(to); t = sched.Next(t) {
		if isDSTFallbackDuplicate(last, t, loc) {
			continue
		}
		last = t
		if t.In(loc).Format("15:04") == wall {
			n++
		}
	}
	return n
}

// isDSTFallbackDuplicate mirrors go-cron's isDSTFallBackDuplicate (cron.go):
// next repeats the wall time of the run just fired, after the clocks went back.
func isDSTFallbackDuplicate(prev, next time.Time, loc *time.Location) bool {
	if prev.IsZero() || next.IsZero() {
		return false
	}
	p, n := prev.In(loc), next.In(loc)
	if p.Year() != n.Year() || p.Month() != n.Month() || p.Day() != n.Day() ||
		p.Hour() != n.Hour() || p.Minute() != n.Minute() || p.Second() != n.Second() {
		return false
	}
	_, pOff := p.Zone()
	_, nOff := n.Zone()
	return nOff < pOff
}

type transition struct {
	at            time.Time
	before, after int // UTC offsets in seconds
}

// transitions finds offset changes in loc between from and to, to the second.
func transitions(loc *time.Location, from, to time.Time) []transition {
	var out []transition
	offset := func(t time.Time) int { _, o := t.In(loc).Zone(); return o }
	for t := from; t.Before(to); t = t.Add(time.Hour) {
		a, b := offset(t), offset(t.Add(time.Hour))
		if a == b {
			continue
		}
		lo, hi := t, t.Add(time.Hour)
		for hi.Sub(lo) > time.Second {
			mid := lo.Add(hi.Sub(lo) / 2)
			if offset(mid) == a {
				lo = mid
			} else {
				hi = mid
			}
		}
		out = append(out, transition{at: hi, before: a, after: b})
	}
	return out
}
