package spec

import (
	"fmt"
	"strconv"
	"strings"
)

var descriptorFields = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

var triggeredOnly = map[string]bool{"@triggered": true, "@manual": true, "@none": true}

var dayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

var monthFull = []string{"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}

var nth = []string{"", "first", "second", "third", "fourth", "fifth"}

// errUndescribed marks a construct the grammar has no phrase for yet.
var errUndescribed = fmt.Errorf("no description for this construct")

// Describe returns an English description of a cron expression in a small
// regular grammar: one clause per field, in a fixed order, each starting with
// a fixed keyword, 24-hour times. The time zone is left out: callers show
// the schedule's location beside it. Call it on expressions go-cron
// accepted. If a construct has no phrase yet, it returns the literal
// per-field rendering and a non-nil error, so callers always have something
// to show.
func Describe(expression string) (string, error) {
	s, err := Fields(expression)
	if err != nil {
		return expression, err
	}
	switch {
	case s.Descriptor == "@every":
		return "every " + s.Every + " from when the schedule starts", nil
	case triggeredOnly[s.Descriptor]:
		return "only when triggered", nil
	case s.Descriptor != "":
		fields, ok := descriptorFields[s.Descriptor]
		if !ok {
			return expression, fmt.Errorf("unknown descriptor %s", s.Descriptor)
		}
		return Describe(fields)
	}
	d, err := describeFields(s)
	if err != nil {
		return literal(s), err
	}
	return d, nil
}

// DescribeNote spells out a hashed step's values in terms of its hashed
// start ("at h, h+15, h+30 and h+45 minutes past the hour, where h is a
// hashed minute from 0 to 14"), so Describe can keep the short "every 15
// minutes from a hashed start". It returns "" when no time field has a
// hashed step or the expression has no description.
func DescribeNote(expression string) string {
	s, err := Fields(expression)
	if err != nil || s.Descriptor != "" {
		return ""
	}
	t, hashed, err := describeTime(s, true)
	if err != nil || !hashed {
		return ""
	}
	return t
}

func literal(s Spec) string {
	parts := make([]string, 0, len(s.Fields))
	for _, f := range s.Fields {
		parts = append(parts, f.Name+" "+f.Text)
	}
	return strings.Join(parts, ", ")
}

func describeFields(s Spec) (string, error) {
	clauses := []string{}
	t, _, err := describeTime(s, false)
	if err != nil {
		return "", err
	}
	clauses = append(clauses, t)
	if d, err := describeDay(s); err != nil {
		return "", err
	} else if d != "" {
		clauses = append(clauses, d)
	}
	if f, ok := s.Field(Month); ok && f.Text != "*" && f.Text != "?" {
		m, err := describeMonth(f.Text)
		if err != nil {
			return "", err
		}
		clauses = append(clauses, m)
	}
	if f, ok := s.Field(Year); ok && f.Text != "*" && f.Text != "?" {
		items, err := simpleItems(f)
		if err != nil {
			return "", err
		}
		clauses = append(clauses, "in "+list(items, strconv.Itoa, fieldBounds[Year]))
	}
	return strings.Join(clauses, ", "), nil
}

// simpleItems parses a field made only of values and ranges.
func simpleItems(f Field) ([]item, error) {
	var out []item
	for _, e := range elements(f.Text) {
		it, perr := parseItem(e, fieldBounds[f.Name])
		if perr != nil || (it.kind != kValue && it.kind != kRange) {
			return nil, errUndescribed
		}
		out = append(out, it)
	}
	return out, nil
}

// list names values and ranges ("9 thru 17"); a range of two adjacent
// values names both ("9 and 10"), as day-of-week ranges do.
func list(items []item, name func(int) string, b bounds) string {
	var words []string
	for _, it := range items {
		switch {
		case it.kind != kRange:
			words = append(words, name(it.lo))
		case it.hi == it.lo+1 || (it.lo == b.max && it.hi == b.min):
			words = append(words, name(it.lo), name(it.hi))
		default:
			words = append(words, name(it.lo)+" thru "+name(it.hi))
		}
	}
	return andList(words)
}

func plural(items []item) bool { return len(items) > 1 || items[0].kind == kRange }

// ---- time: second, minute, hour ----

type unitPhrase struct {
	text     string
	kind     string // "values", "hashed", "step", "every"
	field    string
	nums     string // the values alone ("0, 15, 30 and 45"), for per-unit wording
	note     string // interval note for a list cut short
	where    string // what a hashed step's variable stands for
	hourStep bool   // hours from a step: "of every day" when days are free
}

// describeTime renders the time clause. With expand, hashed steps render as
// their values in terms of the hashed start, and hashed reports whether any did.
func describeTime(s Spec, expand bool) (text string, hashed bool, err error) {
	sec, hasSec := s.Field(Second)
	min, _ := s.Field(Minute)
	hour, _ := s.Field(Hour)

	// a fixed second and up to six minute × hour combinations → clock times
	if mins, ok := fieldRuns(min); ok {
		sv, secOK := 0, !hasSec
		if hasSec {
			sv, secOK = single(sec)
		}
		if hours, ok := fieldRuns(hour); ok && secOK && len(mins)*len(hours) <= 6 {
			var clocks []string
			for _, h := range hours {
				for _, m := range mins {
					c := fmt.Sprintf("%02d:%02d", h, m)
					if sv != 0 {
						c += fmt.Sprintf(":%02d", sv)
					}
					clocks = append(clocks, c)
				}
			}
			return "at " + andList(clocks), false, nil
		}
	}

	fields := []Field{min, hour}
	if hasSec && sec.Text != "0" {
		fields = []Field{sec, min, hour}
	}
	var phrases []unitPhrase
	var notes, wheres []string
	vars := "hkj" // one variable per hashed step
	for _, f := range fields {
		v := ""
		if expand {
			v = vars[len(wheres) : len(wheres)+1]
		}
		p, err := timeUnit(f, v)
		if err != nil {
			return "", false, err
		}
		if p.kind == "every" && len(phrases) > 0 {
			prev := phrases[len(phrases)-1].kind
			if prev == "step" || prev == "every" {
				continue // "every 15 minutes" already covers every hour
			}
		}
		phrases = append(phrases, p)
		if p.note != "" {
			notes = append(notes, p.note)
		}
		if p.where != "" {
			wheres = append(wheres, p.where)
		}
	}
	out := joinTime(phrases)
	if dom, _ := s.Field(DayOfMonth); phrases[len(phrases)-1].hourStep && isStar(dom) {
		if dow, _ := s.Field(DayOfWeek); isStar(dow) {
			out += " of every day"
		}
	}
	if len(wheres) > 0 {
		out += ", where " + strings.Join(wheres, " and ")
	}
	if len(notes) > 0 {
		out += " (" + strings.Join(notes, "; ") + ")"
	}
	return out, len(wheres) > 0, nil
}

// joinTime joins the time phrases, finest first. Seconds or minutes that
// lead with values read per unit ("at 5 and 10 minutes past the hour");
// minute 0 of every hour is "the hour". Other phrases join with "of".
func joinTime(ps []unitPhrase) string {
	join := func(ps []unitPhrase) string {
		parts := make([]string, len(ps))
		for i, p := range ps {
			parts[i] = p.text
		}
		return strings.Join(parts, " of ")
	}
	theHour := func(ps []unitPhrase) bool {
		return len(ps) == 2 && ps[0].field == Minute && ps[0].nums == "0" &&
			ps[1].field == Hour && ps[1].kind == "every"
	}
	first, rest := ps[0], ps[1:]
	if first.nums == "" || first.field == Hour {
		if first.kind == "values" || first.kind == "hashed" {
			return "at " + join(ps)
		}
		return join(ps)
	}
	if theHour(ps) {
		return "on the hour"
	}
	unit := first.field + "s"
	if first.nums == "1" {
		unit = first.field
	}
	out := "at " + first.nums + " " + unit + " past "
	switch {
	case len(rest) == 1 && rest[0].kind == "every":
		return out + "the " + rest[0].field
	case theHour(rest):
		return out + "the hour"
	}
	return out + join(rest)
}

func single(f Field) (int, bool) {
	it, err := parseItem(f.Text, fieldBounds[f.Name])
	if err != nil || it.kind != kValue || strings.Contains(f.Text, ",") {
		return 0, false
	}
	return it.lo, true
}

func isStar(f Field) bool { return f.Text == "*" || f.Text == "?" }

// listLimit is how many values a field lists before cutting the list short.
var listLimit = map[string]int{Second: 6, Minute: 6, Hour: 12, DayOfMonth: 6}

// container is the unit a field starts over in.
var container = map[string]string{Second: "minute", Minute: "hour", Hour: "day", DayOfMonth: "month"}

// timeUnit renders one time field. v names a hashed step's start; "" keeps
// the short "every N <units> from a hashed start".
func timeUnit(f Field, v string) (unitPhrase, error) {
	b := fieldBounds[f.Name]
	sg, pl := f.Name, f.Name+"s"
	els := elements(f.Text)
	if len(els) == 1 {
		it, perr := parseItem(els[0], b)
		if perr != nil {
			return unitPhrase{}, errUndescribed
		}
		switch it.kind {
		case kStar:
			return unitPhrase{text: "every " + sg, kind: "every", field: f.Name}, nil
		case kStep:
			nums, n, note := stepRuns(it, f.Name, sg, pl, strconv.Itoa)
			unit := pl
			if n == 1 {
				unit = sg
			}
			return unitPhrase{text: unit + " " + nums, kind: "values", field: f.Name, nums: nums, note: note, hourStep: f.Name == Hour}, nil
		case kHash:
			return unitPhrase{text: "a hashed " + sg + between(it), kind: "hashed"}, nil
		case kHashStep:
			if v == "" {
				return unitPhrase{text: fmt.Sprintf("every %d %s from a hashed start%s", it.step, pl, between(it)), kind: "step"}, nil
			}
			if it.lo > it.hi {
				return unitPhrase{}, errUndescribed // wrapped hashed range
			}
			unit, nums, where, note := hashStepValues(it, f.Name, sg, pl, v)
			return unitPhrase{text: unit + " " + nums, kind: "values", field: f.Name, nums: nums, where: where, note: note, hourStep: f.Name == Hour}, nil
		}
	}
	items, err := simpleItems(f)
	if err != nil {
		return unitPhrase{}, err
	}
	unit := sg
	if plural(items) {
		unit = pl
	}
	nums := list(items, strconv.Itoa, b)
	return unitPhrase{text: unit + " " + nums, kind: "values", field: f.Name, nums: nums}, nil
}

// stepRuns renders a step as the values it fires on, each through word
// ("0, 15, 30 and 45"), and how many there are. A list cut short gets a
// note with the interval, and says where it starts over when the gap
// across the boundary differs: "every 7 minutes, starting over each hour".
func stepRuns(it item, field, sg, pl string, word func(int) string) (text string, n int, note string) {
	b := fieldBounds[field]
	vals := runs(it, b)
	words := make([]string, len(vals))
	for i, v := range vals {
		words[i] = word(v)
	}
	if len(vals) <= listLimit[field] {
		return andList(words), len(vals), ""
	}
	text = strings.Join(words[:3], ", ") + ", … " + words[len(words)-1]
	note = "every " + sg
	if it.step > 1 {
		note = fmt.Sprintf("every %d %s", it.step, pl)
	}
	if vals[0]+b.max-b.min+1-vals[len(vals)-1] != it.step {
		note += ", starting over each " + container[field]
	}
	return text, len(vals), note
}

// hashStepValues renders H/N as values in terms of the job's hashed first
// value v, which go-cron picks as lo + hash mod N: "minutes h, h+15, h+30
// and h+45", where "h is a hashed minute from 0 to 14". When N does not
// divide the range, the run count depends on v, so the list ends
// "… up to <last allowed value>".
func hashStepValues(it item, field, sg, pl, v string) (unit, nums, where, note string) {
	span, vmax := it.hi-it.lo+1, min(it.lo+it.step-1, it.hi)
	where = fmt.Sprintf("%s is a hashed %s from %d to %d", v, sg, it.lo, vmax)
	term := func(k int) string {
		if k == 0 {
			return v
		}
		return fmt.Sprintf("%s+%d", v, k*it.step)
	}
	if it.step >= span {
		return sg, v, where, ""
	}
	if span%it.step != 0 {
		var words []string
		for k := 0; k < 3 && vmax+k*it.step <= it.hi; k++ {
			words = append(words, term(k))
		}
		note = fmt.Sprintf("every %d %s, starting over each %s", it.step, pl, container[field])
		return pl, strings.Join(words, ", ") + fmt.Sprintf(", … up to %d", it.hi), where, note
	}
	n := span / it.step
	words := make([]string, n)
	for k := range words {
		words[k] = term(k)
	}
	if n <= listLimit[field] {
		return pl, andList(words), where, ""
	}
	note = fmt.Sprintf("every %d %s", it.step, pl)
	return pl, strings.Join(words[:3], ", ") + ", … " + words[n-1], where, note
}

// runs lists the values an element fires on, in firing order. A wrapped
// range (22-2) continues past the field's end, as go-cron does.
func runs(it item, b bounds) []int {
	step, end, size := max(it.step, 1), it.hi, b.max-b.min+1
	if it.lo > it.hi {
		end += size
	}
	var out []int
	for v := it.lo; v <= end; v += step {
		out = append(out, (v-b.min)%size+b.min)
	}
	return out
}

// fieldRuns lists every value a field fires on, in written order; false
// for hashed fields, whose values depend on the job.
func fieldRuns(f Field) ([]int, bool) {
	var out []int
	for _, e := range elements(f.Text) {
		it, perr := parseItem(e, fieldBounds[f.Name])
		if perr != nil || it.kind == kHash || it.kind == kHashStep {
			return nil, false
		}
		out = append(out, runs(it, fieldBounds[f.Name])...)
	}
	return out, true
}

func andList(words []string) string {
	if len(words) == 1 {
		return words[0]
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

func between(it item) string {
	if !it.ranged {
		return ""
	}
	return fmt.Sprintf(" between %d and %d", it.lo, it.hi)
}

// ---- day: day-of-month and day-of-week ----

func describeDay(s Spec) (string, error) {
	dom, _ := s.Field(DayOfMonth)
	dow, _ := s.Field(DayOfWeek)
	domStar := dom.Text == "*" || dom.Text == "?"
	dowStar := dow.Text == "*" || dow.Text == "?"
	switch {
	case domStar && dowStar:
		return "", nil
	case dowStar:
		d, err := domPhrase(dom.Text)
		return "on " + d, err
	case domStar:
		d, err := dowPhrase(dow.Text)
		return "on " + d, err
	}
	d, err := domPhrase(dom.Text)
	if err != nil {
		return "", err
	}
	w, err := dowPhrase(dow.Text)
	if err != nil {
		return "", err
	}
	return "on " + d + " when it falls on " + w, nil
}

func domPhrase(text string) (string, error) {
	up := strings.ToUpper(text)
	switch {
	case up == "L":
		return "the last day of the month", nil
	case up == "LW":
		return "the last weekday of the month", nil
	case strings.HasPrefix(up, "L-") && !strings.Contains(up, ","):
		n, err := strconv.Atoi(up[2:])
		if err != nil {
			return "", errUndescribed
		}
		return "the " + ordinal(n) + " day before the end of the month", nil
	case strings.HasSuffix(up, "W") && !strings.Contains(up, ","):
		n, err := strconv.Atoi(up[:len(up)-1])
		if err != nil {
			return "", errUndescribed
		}
		return "the weekday nearest the " + ordinal(n), nil
	}
	f := Field{Name: DayOfMonth, Text: text}
	if els := elements(text); len(els) == 1 {
		it, perr := parseItem(els[0], fieldBounds[DayOfMonth])
		if perr != nil {
			return "", errUndescribed
		}
		switch it.kind {
		case kStep:
			days, _, note := stepRuns(it, DayOfMonth, "day", "days", ordinal)
			text := "the " + days + " of every month"
			if note != "" {
				text += " (" + note + ")"
			}
			return text, nil
		case kHash:
			return "a hashed day of the month" + between(it), nil
		case kHashStep:
			return "", errUndescribed
		}
	}
	if els := elements(text); len(els) > 1 && strings.ContainsAny(up, "LW") {
		var words []string
		for _, e := range els {
			w, err := domItem(e)
			if err != nil {
				return "", err
			}
			words = append(words, w...)
		}
		return andList(words) + " of the month", nil
	}
	items, err := simpleItems(f)
	if err != nil {
		return "", err
	}
	return "the " + list(items, ordinal, fieldBounds[DayOfMonth]) + " of the month", nil
}

// domItem renders one element of a day-of-month list that holds L or W
// forms; the caller adds "of the month". A range of two adjacent days is
// two entries ("the 1st", "the 2nd").
func domItem(e string) ([]string, error) {
	up := strings.ToUpper(e)
	switch {
	case up == "L":
		return []string{"the last day"}, nil
	case up == "LW":
		return []string{"the last weekday"}, nil
	case strings.HasPrefix(up, "L-"):
		n, err := strconv.Atoi(up[2:])
		if err != nil {
			return nil, errUndescribed
		}
		return []string{"the " + ordinal(n) + " day before the end"}, nil
	case strings.HasSuffix(up, "W"):
		n, err := strconv.Atoi(up[:len(up)-1])
		if err != nil {
			return nil, errUndescribed
		}
		return []string{"the weekday nearest the " + ordinal(n)}, nil
	}
	it, perr := parseItem(e, fieldBounds[DayOfMonth])
	switch {
	case perr != nil:
		return nil, errUndescribed
	case it.kind == kValue:
		return []string{"the " + ordinal(it.lo)}, nil
	case it.kind == kRange && it.hi == it.lo+1:
		return []string{"the " + ordinal(it.lo), "the " + ordinal(it.hi)}, nil
	case it.kind == kRange:
		return []string{"the " + ordinal(it.lo) + " thru " + ordinal(it.hi)}, nil
	}
	return nil, errUndescribed
}

func dowPhrase(text string) (string, error) {
	els := elements(text)
	if len(els) == 1 {
		it, perr := parseItem(els[0], fieldBounds[DayOfWeek])
		if perr == nil && it.kind == kHash && !it.ranged {
			return "a hashed day of the week", nil
		}
		if perr == nil && it.kind == kStep {
			var days []string
			for _, d := range runs(it, fieldBounds[DayOfWeek]) {
				if d == 7 && len(days) > 0 && days[0] == dayNames[0] {
					continue // 0 and 7 are both Sunday
				}
				days = append(days, dayNames[d])
			}
			return andList(days), nil
		}
	}
	var words []string
	for _, e := range els {
		w, err := dowItem(e)
		if err != nil {
			return "", err
		}
		words = append(words, w...)
	}
	return andList(words), nil
}

// dowItem renders one day-of-week list element: a day, a range of days
// ("Monday thru Friday"; two adjacent days are two list entries, "Monday"
// and "Tuesday"), or an nth day of the month ("the second Sunday").
func dowItem(e string) ([]string, error) {
	b := fieldBounds[DayOfWeek]
	if day, n, ok := strings.Cut(e, "#"); ok {
		d, perr := parseNum(day, b)
		if perr != nil {
			return nil, errUndescribed
		}
		if strings.EqualFold(n, "L") {
			return []string{"the last " + dayNames[d]}, nil
		}
		k, err := strconv.Atoi(n)
		if err != nil || k < 1 || k > 5 {
			return nil, errUndescribed
		}
		return []string{"the " + nth[k] + " " + dayNames[d]}, nil
	}
	it, perr := parseItem(e, b)
	if perr != nil || (it.kind != kValue && it.kind != kRange) {
		return nil, errUndescribed
	}
	if it.kind == kValue {
		return []string{dayNames[it.lo]}, nil
	}
	if (it.hi-it.lo+7)%7 == 1 { // adjacent, also across the week's end (SAT-SUN)
		return []string{dayNames[it.lo], dayNames[it.hi]}, nil
	}
	return []string{dayNames[it.lo] + " thru " + dayNames[it.hi]}, nil
}

// ---- month ----

func describeMonth(text string) (string, error) {
	name := func(n int) string { return monthFull[n] }
	if els := elements(text); len(els) == 1 {
		it, perr := parseItem(els[0], fieldBounds[Month])
		if perr != nil {
			return "", errUndescribed
		}
		switch it.kind {
		case kStep:
			var names []string
			for _, m := range runs(it, fieldBounds[Month]) {
				names = append(names, name(m))
			}
			return "in " + andList(names), nil
		case kHash:
			if it.ranged {
				return "", errUndescribed
			}
			return "in a hashed month", nil
		case kHashStep:
			return "", errUndescribed
		}
	}
	items, err := simpleItems(Field{Name: Month, Text: text})
	if err != nil {
		return "", err
	}
	return "in " + list(items, name, fieldBounds[Month]), nil
}

func ordinal(n int) string {
	suffix := "th"
	switch {
	case n%100 >= 11 && n%100 <= 13:
	case n%10 == 1:
		suffix = "st"
	case n%10 == 2:
		suffix = "nd"
	case n%10 == 3:
		suffix = "rd"
	}
	return strconv.Itoa(n) + suffix
}
