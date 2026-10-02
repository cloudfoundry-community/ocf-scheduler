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
// a fixed keyword, 24-hour times. Call it on expressions go-cron accepted. If
// a construct has no phrase yet, it returns the literal per-field rendering
// and a non-nil error, so callers always have something to show.
func Describe(expression string) (string, error) {
	s, err := Fields(expression)
	if err != nil {
		return expression, err
	}
	zone := ""
	if s.Location != "" {
		zone = ", " + s.Location + " time"
	}
	switch {
	case s.Descriptor == "@every":
		return "every " + s.Every + " from when the schedule starts" + zone, nil
	case triggeredOnly[s.Descriptor]:
		return "only when triggered", nil
	case s.Descriptor != "":
		fields, ok := descriptorFields[s.Descriptor]
		if !ok {
			return expression, fmt.Errorf("unknown descriptor %s", s.Descriptor)
		}
		d, err := Describe(fields)
		return d + zone, err
	}
	d, err := describeFields(s)
	if err != nil {
		return literal(s), err
	}
	return d + zone, nil
}

func literal(s Spec) string {
	parts := make([]string, 0, len(s.Fields)+1)
	for _, f := range s.Fields {
		parts = append(parts, f.Name+" "+f.Text)
	}
	if s.Location != "" {
		parts = append(parts, s.Location+" time")
	}
	return strings.Join(parts, ", ")
}

func describeFields(s Spec) (string, error) {
	clauses := []string{}
	t, err := describeTime(s)
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
		clauses = append(clauses, "in "+list(items, strconv.Itoa))
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

func list(items []item, name func(int) string) string {
	words := make([]string, len(items))
	for i, it := range items {
		words[i] = name(it.lo)
		if it.kind == kRange {
			words[i] += " to " + name(it.hi)
		}
	}
	if len(words) == 1 {
		return words[0]
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

func plural(items []item) bool { return len(items) > 1 || items[0].kind == kRange }

// ---- time: second, minute, hour ----

type unitPhrase struct {
	text string
	kind string // "values", "hashed", "step", "every"
}

func describeTime(s Spec) (string, error) {
	sec, hasSec := s.Field(Second)
	min, _ := s.Field(Minute)
	hour, _ := s.Field(Hour)

	// all single values → a clock time
	if h, ok := single(hour); ok {
		if m, ok := single(min); ok {
			if !hasSec {
				return fmt.Sprintf("at %02d:%02d", h, m), nil
			}
			if sv, ok := single(sec); ok {
				if sv == 0 {
					return fmt.Sprintf("at %02d:%02d", h, m), nil
				}
				return fmt.Sprintf("at %02d:%02d:%02d", h, m, sv), nil
			}
		}
	}

	fields := []Field{min, hour}
	if hasSec && sec.Text != "0" {
		fields = []Field{sec, min, hour}
	}
	var phrases []unitPhrase
	for _, f := range fields {
		p, err := timeUnit(f)
		if err != nil {
			return "", err
		}
		if p.kind == "every" && len(phrases) > 0 {
			prev := phrases[len(phrases)-1].kind
			if prev == "step" || prev == "every" {
				continue // "every 15 minutes" already covers every hour
			}
		}
		phrases = append(phrases, p)
	}
	parts := make([]string, len(phrases))
	for i, p := range phrases {
		parts[i] = p.text
	}
	out := strings.Join(parts, " of ")
	if phrases[0].kind == "values" || phrases[0].kind == "hashed" {
		out = "at " + out
	}
	return out, nil
}

func single(f Field) (int, bool) {
	it, err := parseItem(f.Text, fieldBounds[f.Name])
	if err != nil || it.kind != kValue || strings.Contains(f.Text, ",") {
		return 0, false
	}
	return it.lo, true
}

func timeUnit(f Field) (unitPhrase, error) {
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
			return unitPhrase{"every " + sg, "every"}, nil
		case kStep:
			return unitPhrase{stepPhrase(it, sg, pl, strconv.Itoa), "step"}, nil
		case kHash:
			return unitPhrase{"a hashed " + sg + between(it), "hashed"}, nil
		case kHashStep:
			return unitPhrase{fmt.Sprintf("every %d %s from a hashed start%s", it.step, pl, between(it)), "step"}, nil
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
	return unitPhrase{unit + " " + list(items, strconv.Itoa), "values"}, nil
}

func stepPhrase(it item, sg, pl string, name func(int) string) string {
	out := "every " + sg
	if it.step > 1 {
		out = fmt.Sprintf("every %d %s", it.step, pl)
	}
	if it.ranged {
		out += " from " + name(it.lo) + " to " + name(it.hi)
	}
	return out
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
		return fmt.Sprintf("the weekday nearest day %d", n), nil
	}
	f := Field{Name: DayOfMonth, Text: text}
	if els := elements(text); len(els) == 1 {
		it, perr := parseItem(els[0], fieldBounds[DayOfMonth])
		if perr != nil {
			return "", errUndescribed
		}
		switch it.kind {
		case kStep:
			return stepPhrase(it, "day", "days", strconv.Itoa) + " of the month", nil
		case kHash:
			return "a hashed day of the month" + between(it), nil
		case kHashStep:
			return "", errUndescribed
		}
	}
	items, err := simpleItems(f)
	if err != nil {
		return "", err
	}
	unit := "day"
	if plural(items) {
		unit = "days"
	}
	return unit + " " + list(items, strconv.Itoa) + " of the month", nil
}

func dowPhrase(text string) (string, error) {
	if day, n, ok := strings.Cut(text, "#"); ok && !strings.Contains(text, ",") {
		d, perr := parseNum(day, fieldBounds[DayOfWeek])
		if perr != nil {
			return "", errUndescribed
		}
		if strings.EqualFold(n, "L") {
			return "the last " + dayNames[d], nil
		}
		k, err := strconv.Atoi(n)
		if err != nil || k < 1 || k > 5 {
			return "", errUndescribed
		}
		return "the " + nth[k] + " " + dayNames[d], nil
	}
	f := Field{Name: DayOfWeek, Text: text}
	if els := elements(text); len(els) == 1 {
		if it, perr := parseItem(els[0], fieldBounds[DayOfWeek]); perr == nil && it.kind == kHash && !it.ranged {
			return "a hashed day of the week", nil
		}
	}
	items, err := simpleItems(f)
	if err != nil {
		return "", err
	}
	return list(items, func(n int) string { return dayNames[n] }), nil
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
			return "in " + stepPhrase(it, "month", "months", name), nil
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
	return "in " + list(items, name), nil
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
