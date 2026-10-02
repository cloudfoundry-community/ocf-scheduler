// Package spec reads cron expressions the way go-cron's FullParser does, to
// describe them in English and to explain why one failed to parse. It depends
// only on the standard library so it can move into go-cron unchanged.
package spec

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Field names, in cron order.
const (
	Second     = "second"
	Minute     = "minute"
	Hour       = "hour"
	DayOfMonth = "day-of-month"
	Month      = "month"
	DayOfWeek  = "day-of-week"
	Year       = "year"
)

// Field is one whitespace-separated field of a cron expression.
type Field struct {
	Name   string
	Text   string
	Offset int // byte offset of Text in the expression
}

// Spec is a cron expression split the way go-cron's FullParser splits it.
type Spec struct {
	Expression string
	Location   string // value of a TZ= or CRON_TZ= prefix, quotes removed
	Descriptor string // "@daily", "@every", …; empty for field expressions
	Every      string // the duration text of "@every <duration>"
	Fields     []Field
}

// Field returns the named field and whether the expression has it.
func (s Spec) Field(name string) (Field, bool) {
	for _, f := range s.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}

type token struct {
	text   string
	offset int
}

func split(s string) []token {
	var out []token
	start := -1
	for i, r := range s {
		switch {
		case unicode.IsSpace(r) && start >= 0:
			out = append(out, token{s[start:i], start})
			start = -1
		case !unicode.IsSpace(r) && start < 0:
			start = i
		}
	}
	if start >= 0 {
		out = append(out, token{s[start:], start})
	}
	return out
}

// Fields splits expression into its time zone, descriptor and named fields.
// It applies FullParser's field-count rules: 5 fields are minute to
// day-of-week; 6 fields start with seconds unless the last field looks like a
// year (a number >= 100); 7 fields are seconds to year.
func Fields(expression string) (Spec, error) {
	s := Spec{Expression: expression}
	toks := split(expression)
	if len(toks) == 0 {
		return s, fmt.Errorf("empty expression")
	}
	for _, prefix := range []string{"CRON_TZ=", "TZ="} {
		if strings.HasPrefix(toks[0].text, prefix) {
			s.Location = unquote(toks[0].text[len(prefix):])
			toks = toks[1:]
			break
		}
	}
	if len(toks) == 0 {
		return s, fmt.Errorf("no schedule after the time zone")
	}
	if strings.HasPrefix(toks[0].text, "@") {
		s.Descriptor = toks[0].text
		if s.Descriptor == "@every" {
			if len(toks) != 2 {
				return s, fmt.Errorf("@every takes one duration")
			}
			s.Every = toks[1].text
		} else if len(toks) != 1 {
			return s, fmt.Errorf("unexpected text after %s", s.Descriptor)
		}
		return s, nil
	}
	var names []string
	switch {
	case len(toks) == 5:
		names = []string{Minute, Hour, DayOfMonth, Month, DayOfWeek}
	case len(toks) == 6 && looksLikeYear(toks[5].text):
		names = []string{Minute, Hour, DayOfMonth, Month, DayOfWeek, Year}
	case len(toks) == 6:
		names = []string{Second, Minute, Hour, DayOfMonth, Month, DayOfWeek}
	case len(toks) == 7:
		names = []string{Second, Minute, Hour, DayOfMonth, Month, DayOfWeek, Year}
	default:
		return s, fmt.Errorf("expected 5 to 7 fields, found %d", len(toks))
	}
	for i, t := range toks {
		s.Fields = append(s.Fields, Field{Name: names[i], Text: t.text, Offset: t.offset})
	}
	return s, nil
}

func unquote(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	return v
}

// looksLikeYear mirrors go-cron: any number >= 100 in the field.
func looksLikeYear(field string) bool {
	for _, part := range strings.FieldsFunc(field, func(r rune) bool {
		return r == ',' || r == '-' || r == '/'
	}) {
		if n, err := strconv.Atoi(part); err == nil && n >= 100 {
			return true
		}
	}
	return false
}
