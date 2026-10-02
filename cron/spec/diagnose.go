package spec

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseError explains why go-cron rejected an expression, naming the field
// and value where possible (the shape proposed in go-cron #298).
type ParseError struct {
	Expression string
	Field      string // "minute", "day-of-week", "time zone", …; "" if not field-specific
	Value      string // the offending text
	Reason     string // "is out of range", "has a step below 1", …
	ValidRange string // "0-59", …; "" if not applicable
	Offset     int    // byte offset of Value in Expression; -1 if unknown
}

// Message is the explanation without the expression.
func (e *ParseError) Message() string {
	if e.Field == "" {
		return e.Reason
	}
	msg := fmt.Sprintf("%s field value %s %s", e.Field, e.Value, e.Reason)
	if e.ValidRange != "" {
		msg += " (valid: " + e.ValidRange + ")"
	}
	return msg
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("invalid cron expression %q: %s", e.Expression, e.Message())
}

var descriptors = "@yearly, @annually, @monthly, @weekly, @daily, @midnight, @hourly, @every <duration>"

// Diagnose explains parseErr, the error go-cron returned for expression. It
// only explains: if it finds no cause, the Reason is go-cron's own message.
func Diagnose(expression string, parseErr error) *ParseError {
	fallback := &ParseError{Expression: expression, Offset: -1}
	if parseErr != nil {
		fallback.Reason = parseErr.Error()
	}
	s, err := Fields(expression)
	if err != nil {
		fallback.Reason = err.Error()
		return fallback
	}
	if s.Location != "" {
		if _, err := time.LoadLocation(s.Location); err != nil {
			return &ParseError{Expression: expression, Field: "time zone", Value: s.Location,
				Reason: "is not a known time zone", Offset: strings.Index(expression, s.Location)}
		}
	}
	if s.Descriptor != "" {
		off := strings.Index(expression, s.Descriptor)
		switch {
		case s.Descriptor == "@every":
			d, err := time.ParseDuration(s.Every)
			if err != nil || d < time.Second {
				return &ParseError{Expression: expression, Field: "@every", Value: s.Every,
					Reason: "is not a duration of at least 1s", Offset: strings.LastIndex(expression, s.Every)}
			}
		case descriptorFields[s.Descriptor] == "" && !triggeredOnly[s.Descriptor]:
			return &ParseError{Expression: expression, Field: "descriptor", Value: s.Descriptor,
				Reason: "is not a descriptor", ValidRange: descriptors, Offset: off}
		}
		return fallback
	}
	for _, f := range s.Fields {
		off := f.Offset
		for _, e := range elements(f.Text) {
			if e == "" {
				off++
				continue
			}
			if ie := checkElement(f.Name, e); ie != nil {
				at := off
				if i := strings.Index(e, ie.value); i >= 0 {
					at += i
				}
				return &ParseError{Expression: expression, Field: f.Name, Value: ie.value,
					Reason: ie.reason, ValidRange: ie.valid, Offset: at}
			}
			off += len(e) + 1
		}
	}
	return fallback
}

func checkElement(field, e string) *itemError {
	up := strings.ToUpper(e)
	switch {
	case field == DayOfMonth && (up == "L" || up == "LW"):
		return nil
	case field == DayOfMonth && strings.HasPrefix(up, "L-"):
		if n, err := strconv.Atoi(up[2:]); err != nil || n < 1 || n > 30 {
			return &itemError{e, "has an offset outside 1-30", ""}
		}
		return nil
	case field == DayOfMonth && strings.HasSuffix(up, "W"):
		if n, err := strconv.Atoi(up[:len(up)-1]); err != nil || n < 1 || n > 31 {
			return &itemError{e, "needs a day 1-31 before W", ""}
		}
		return nil
	case field == DayOfWeek && strings.Contains(e, "#"):
		day, n, _ := strings.Cut(e, "#")
		if _, ie := parseNum(day, fieldBounds[DayOfWeek]); ie != nil {
			return ie
		}
		if k, err := strconv.Atoi(n); !strings.EqualFold(n, "L") && (err != nil || k < 1 || k > 5) {
			return &itemError{e, "needs an occurrence 1-5 or L after #", ""}
		}
		return nil
	case field == Year && strings.HasPrefix(e, "H"):
		return &itemError{e, "cannot be hashed", ""}
	}
	_, ie := parseItem(e, fieldBounds[field])
	return ie
}
