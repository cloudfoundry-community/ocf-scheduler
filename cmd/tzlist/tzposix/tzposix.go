package tzposix

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// tzRegex is compiled once at package level for POSIX TZ string parsing.
// Captures: StdName, StdOffset, DstName (optional), DstOffset (optional),
// StartRule (optional), EndRule (optional)
var tzRegex = regexp.MustCompile(
	`^(?<StdName>[[:alpha:]]{3,}|<[[:alnum:]+-]+>)` +
		`(?<StdOffset>[-+]?[0-9]+(?::[0-9]+){0,2})` +
		`(?<DstName>[[:alpha:]]{3,}|<[[:alnum:]+-]+>)?` +
		`(?<DstOffset>[-+]?[0-9]+(?::[0-9]+){0,2})?` +
		`,?(?<StartRule>(?:J?[0-9]+|M[0-9]+(?:\.[0-9]+){0,2})(?:/[+-]?[0-9]+(?::[0-9]+){0,2})?)?` +
		`,?(?<EndRule>(?:J?[0-9]+|M[0-9]+(?:\.[0-9]+){0,2})(?:/[+-]?[0-9]+(?::[0-9]+){0,2})?)?$`)

// parsedTZ holds the parsed components of a POSIX TZ string.
type parsedTZ struct {
	stdAbbr   string
	stdOffset int
	dstAbbr   string
	dstOffset int
	startRule string
	endRule   string
	hasDST    bool
}

// parseTZ parses a POSIX TZ string into its components.
func parseTZ(posixTZ string) (*parsedTZ, error) {
	matches := tzRegex.FindStringSubmatch(posixTZ)
	if matches == nil {
		return nil, fmt.Errorf("invalid POSIX TZ string format: %s", posixTZ)
	}

	stdOffsetStr := matches[2]
	dstOffsetStr := matches[4]

	stdOffset, err := parseOffset(stdOffsetStr)
	if err != nil {
		return nil, fmt.Errorf("invalid standard offset: %w", err)
	}

	result := &parsedTZ{
		stdAbbr:   matches[1],
		stdOffset: stdOffset,
		dstAbbr:   matches[3],
		startRule: matches[5],
		endRule:   matches[6],
		hasDST:    matches[3] != "",
	}

	if !result.hasDST {
		return result, nil
	}

	// POSIX default: DST is 1 hour ahead of standard time
	result.dstOffset = stdOffset - 3600
	if dstOffsetStr != "" {
		parsedDstOffset, err := parseOffset(dstOffsetStr)
		if err != nil {
			return nil, fmt.Errorf("invalid daylight offset: %w", err)
		}
		result.dstOffset = parsedDstOffset
	}

	return result, nil
}

func DecodeTZ(posixTZ string) (string, string, string, error) {
	tz, err := parseTZ(posixTZ)
	if err != nil {
		return "", "", "", err
	}

	stdDesc := fmt.Sprintf("%s (UTC%s)", tz.stdAbbr, formatOffset(tz.stdOffset))
	if !tz.hasDST {
		return stdDesc, "", "", nil
	}

	dstDesc := fmt.Sprintf("%s (UTC%s)", tz.dstAbbr, formatOffset(tz.dstOffset))

	rulesDesc := ""
	if tz.startRule != "" && tz.endRule != "" {
		rulesDesc = fmt.Sprintf("Starts %s, Ends %s", parseRule(tz.startRule), parseRule(tz.endRule))
	}

	return stdDesc, dstDesc, rulesDesc, nil
}

// HumanReadableTZ parses a POSIX TZ string and returns a human-readable description.
// It handles a common format like "EST5EDT,M3.2.0/02:00:00,M11.1.0/02:00:00"
func HumanReadableTZ(posixTZ string) (string, error) {
	tz, err := parseTZ(posixTZ)
	if err != nil {
		return "", err
	}

	stdDesc := fmt.Sprintf("Standard Time: %s (UTC%s)", tz.stdAbbr, formatOffset(tz.stdOffset))

	if !tz.hasDST {
		return stdDesc + "\n(No Daylight Saving Time rules)", nil
	}

	if !((tz.startRule == "" && tz.endRule == "") || (tz.startRule != "" && tz.endRule != "")) {
		fmt.Fprintln(os.Stderr, "Warning Stand alone TZ rule exists", posixTZ)
	}

	dstDesc := fmt.Sprintf("Daylight Time: %s (UTC%s)", tz.dstAbbr, formatOffset(tz.dstOffset))

	rulesDesc := ""
	if tz.startRule != "" && tz.endRule != "" {
		rulesDesc = fmt.Sprintf("\nRules: Starts %s, Ends %s", parseRule(tz.startRule), parseRule(tz.endRule))
	}

	return fmt.Sprintf("%s\n%s%s", stdDesc, dstDesc, rulesDesc), nil
}

// parseOffset converts a POSIX offset string (e.g., "5", "-10:30") to seconds west of UTC
func parseOffset(offsetStr string) (int, error) {
	// POSIX offsets are West of Greenwich, opposite of ISO 8601
	// "EST5" means 5 hours West of UTC (UTC+5 if we follow standard notation)

	sign := 1
	if after, ok := strings.CutPrefix(offsetStr, "+"); ok {
		offsetStr = after
	} else if after, ok := strings.CutPrefix(offsetStr, "-"); ok {
		offsetStr = after
		sign = -1
	}

	parts := strings.Split(offsetStr, ":")
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	minutes := 0
	if len(parts) > 1 {
		minutes, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
	}
	seconds := 0
	if len(parts) > 2 {
		seconds, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, err
		}
	}
	return sign * (hours*3600 + minutes*60 + seconds), nil
}

// formatOffset converts seconds offset to " +H:M" or " -H:M" string
func formatOffset(offsetSeconds int) string {
	sign := "+"
	if offsetSeconds > 0 {
		sign = "-" // POSIX is backwards, so >0 seconds is actually UTC-X
		offsetSeconds = -offsetSeconds
	}
	// Absolute value for calculation
	absOffset := offsetSeconds
	if absOffset < 0 {
		absOffset = -absOffset
	}

	hours := absOffset / 3600
	minutes := (absOffset % 3600) / 60
	seconds := (absOffset % 3600) % 60
	if seconds != 0 {
		return fmt.Sprintf(" %s%02d:%02d:%02d", sign, hours, minutes, seconds)
	}
	return fmt.Sprintf(" %s%02d:%02d", sign, hours, minutes)
}

// parseRule converts a POSIX rule string (e.g., "M3.2.0/02:00:00") to a description
func parseRule(rule string) string {
	if after, ok := strings.CutPrefix(rule, "M"); ok {
		// Month.Week.Day format
		parts := strings.Split(after, ".")
		if len(parts) >= 3 {
			month := parts[0]
			week := parts[1]
			day := parts[2]
			timeStr := ""
			hours := 2
			minutes := 0
			seconds := 0
			if strings.Contains(day, "/") {
				timeParts := strings.Split(day, "/")
				day = timeParts[0]
				timeStr = timeParts[1]
				if timeStr == "50" {
					hours = 0
					minutes = 50 //   /50: The local time at which the change occurs. If minutes and seconds
					//   are omitted, it defaults to two digits (50 minutes past midnight, 00:50).
					seconds = 0 //   timezones Asia/Gaza Asia/Hebron
				} else if timeStr != "" {
					tparts := strings.Split(timeStr, ":")
					if len(tparts) > 0 {
						hours = atoi(tparts[0])
					}
					if len(tparts) > 1 {
						minutes = atoi(tparts[1])
					}
					if len(tparts) > 2 {
						seconds = atoi(tparts[2])
					}
				}
			}
			if hours == 26 {
				// TODO Do we to support other versions of /26?  Not at this time

				return "on the Friday on or after March 23rd at 02:00:00"
				// /26: The time at which the change occurs, which is 26:00
				// (equivalent to 02:00 on the first Friday on or after March 23).
				// Clocks "spring forward" at this time.
				// timezones Asia/Jerusalem M3.4.4/26
				// March has 31 days the date cannot be expressed with the normal interpretation
				// since it not consistently the 4th week or last week

			} else if hours == 24 {
				timeStr = "midnight of the next day"
			} else {
				t := time.Date(0, 0, 0, hours, minutes, seconds, 0, time.UTC)
				timeStr = t.Format("15:04:05")
			}

			// Mapping basic values to human terms
			months := []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
			weekDesc := map[string]string{"1": "first", "2": "second", "3": "third", "4": "fourth", "5": "last"}
			dayDesc := map[string]string{"0": "Sunday", "1": "Monday", "2": "Tuesday", "3": "Wednesday", "4": "Thursday", "5": "Friday", "6": "Saturday"}

			return fmt.Sprintf("on the %s %s of %s at %s", weekDesc[week], dayDesc[day], months[atoi(month)-1], timeStr)
		}
	} else if after, ok := strings.CutPrefix(rule, "J"); ok {
		julianDay := after
		if julianDay == "365/25" {
			return "at the end of the year"
		}
		return fmt.Sprintf("on Julian Day %s", julianDay)
	} else {
		if rule == "0/0" {
			return "from the start of the year"
		}
		return fmt.Sprintf("on Julian Day %s", rule)
	}

	// Handle Julian day or other formats as needed
	return fmt.Sprintf("Rule: %s", rule)
}

func atoi(s string) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return 0
}
