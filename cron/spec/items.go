package spec

import (
	"fmt"
	"strconv"
	"strings"
)

type bounds struct {
	min, max int
	names    map[string]int
	wrap     bool
}

var monthNames = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

var dowNames = map[string]int{
	"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
}

// fieldBounds mirrors go-cron's bounds (spec.go): every field but year wraps.
var fieldBounds = map[string]bounds{
	Second:     {0, 59, nil, true},
	Minute:     {0, 59, nil, true},
	Hour:       {0, 23, nil, true},
	DayOfMonth: {1, 31, nil, true},
	Month:      {1, 12, monthNames, true},
	DayOfWeek:  {0, 7, dowNames, true},
	Year:       {1, 1<<31 - 1, nil, false},
}

func (b bounds) String() string {
	if b.max == 1<<31-1 {
		return ""
	}
	return fmt.Sprintf("%d-%d", b.min, b.max)
}

type kind int

const (
	kStar     kind = iota // * or ?
	kValue                // 5
	kRange                // 5-10
	kStep                 // */15, 5-50/15, 5/15
	kHash                 // H, H(15-45)
	kHashStep             // H/10, H(15-45)/10
)

// item is one comma-separated element of a field.
type item struct {
	kind   kind
	lo, hi int
	step   int
	ranged bool // a step or hash with its own range (not the whole field)
}

// itemError is a field element that go-cron will reject.
type itemError struct {
	value, reason string
	valid         string
}

func (e *itemError) Error() string { return e.value + " " + e.reason }

func parseItem(t string, b bounds) (item, *itemError) {
	if t == "*" || t == "?" {
		return item{kind: kStar, lo: b.min, hi: b.max}, nil
	}
	if strings.HasPrefix(t, "H") {
		return parseHash(t, b)
	}
	base, stepText, hasStep := strings.Cut(t, "/")
	it := item{kind: kValue}
	switch lo, hi, isRange := strings.Cut(base, "-"); {
	case base == "*":
		it.lo, it.hi = b.min, b.max
	case isRange:
		var err *itemError
		if it.lo, err = parseNum(lo, b); err != nil {
			return it, err
		}
		if it.hi, err = parseNum(hi, b); err != nil {
			return it, err
		}
		if it.lo > it.hi && !b.wrap {
			return it, &itemError{base, "starts after it ends", b.String()}
		}
		it.kind, it.ranged = kRange, true
	default:
		n, err := parseNum(base, b)
		if err != nil {
			return it, err
		}
		it.lo, it.hi = n, n
		if hasStep {
			it.hi, it.ranged = b.max, true
		}
	}
	if hasStep {
		n, err := strconv.Atoi(stepText)
		if err != nil {
			return it, &itemError{t, "has a step that is not a number", ""}
		}
		if n < 1 {
			return it, &itemError{t, "has a step below 1", ""}
		}
		it.kind, it.step = kStep, n
	}
	return it, nil
}

func parseHash(t string, b bounds) (item, *itemError) {
	base, stepText, hasStep := strings.Cut(t, "/")
	it := item{kind: kHash, lo: b.min, hi: b.max}
	if base != "H" {
		inner, ok := strings.CutPrefix(base, "H(")
		inner, ok2 := strings.CutSuffix(inner, ")")
		lo, hi, ok3 := strings.Cut(inner, "-")
		if !ok || !ok2 || !ok3 {
			return it, &itemError{t, "is not H, H(a-b), H/n or H(a-b)/n", ""}
		}
		var err *itemError
		if it.lo, err = parseNum(lo, b); err != nil {
			return it, err
		}
		if it.hi, err = parseNum(hi, b); err != nil {
			return it, err
		}
		it.ranged = true
	}
	if hasStep {
		n, err := strconv.Atoi(stepText)
		if err != nil {
			return it, &itemError{t, "has a step that is not a number", ""}
		}
		if n < 1 {
			return it, &itemError{t, "has a step below 1", ""}
		}
		it.kind, it.step = kHashStep, n
	}
	return it, nil
}

func parseNum(s string, b bounds) (int, *itemError) {
	n, err := strconv.Atoi(s)
	if err != nil {
		v, ok := b.names[strings.ToLower(s)]
		if !ok {
			return 0, &itemError{s, "is not a number" + nameHint(b), b.String()}
		}
		return v, nil
	}
	if n < b.min || n > b.max {
		return 0, &itemError{s, "is out of range", b.String()}
	}
	return n, nil
}

func nameHint(b bounds) string {
	if b.names == nil {
		return ""
	}
	return " or name"
}

// elements splits a field on commas, keeping empty elements (go-cron drops
// them silently).
func elements(text string) []string { return strings.Split(text, ",") }
