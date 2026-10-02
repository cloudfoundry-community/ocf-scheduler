package spec

import (
	"regexp"
	"strings"
	"testing"

	cron "github.com/netresearch/go-cron"
)

var golden = []struct{ expr, want string }{
	{"*/15 * * * *", "every 15 minutes"},
	{"* * * * *", "every minute"},
	{"* * * * * *", "every second"},
	{"*/10 * * * * *", "every 10 seconds"},
	{"0 30 2 * * MON-FRI", "at 02:30, on Monday to Friday"},
	{"15 30 2 * * *", "at 02:30:15"},
	{"0 9-17 * * MON-FRI", "at minute 0 of hours 9 to 17, on Monday to Friday"},
	{"5,10,40 * * * *", "at minutes 5, 10 and 40 of every hour"},
	{"* 2 * * *", "every minute of hour 2"},
	{"30 */2 * * *", "at minute 30 of every 2 hours"},
	{"*/15 9-17 * * *", "every 15 minutes of hours 9 to 17"},
	{"0 22-2 * * *", "at minute 0 of hours 22 to 2"},
	{"5-50/15 * * * *", "every 15 minutes from 5 to 50"},
	{"H(15-45) 2 * * *", "at a hashed minute between 15 and 45 of hour 2"},
	{"H H * * *", "at a hashed minute of a hashed hour"},
	{"H/15 * * * *", "every 15 minutes from a hashed start"},
	{"0 0 13 * FRI", "at 00:00, on day 13 of the month when it falls on Friday"},
	{"0 0 1,15 * *", "at 00:00, on days 1 and 15 of the month"},
	{"0 0 L * *", "at 00:00, on the last day of the month"},
	{"0 0 L-3 * *", "at 00:00, on the 3rd day before the end of the month"},
	{"0 0 LW * *", "at 00:00, on the last weekday of the month"},
	{"0 0 15W * *", "at 00:00, on the weekday nearest day 15"},
	{"0 0 * * FRI#3", "at 00:00, on the third Friday"},
	{"0 0 * * 5#L", "at 00:00, on the last Friday"},
	{"0 0 * * 0,7", "at 00:00, on Sunday and Sunday"},
	{"0 0 1 1 * 2027", "at 00:00, on day 1 of the month, in January, in 2027"},
	{"0 0 1 */3 *", "at 00:00, on day 1 of the month, in every 3 months"},
	{"0 0 1 JAN-MAR *", "at 00:00, on day 1 of the month, in January to March"},
	{"@daily", "at 00:00"},
	{"@hourly", "at minute 0 of every hour"},
	{"@weekly", "at 00:00, on Sunday"},
	{"@yearly", "at 00:00, on day 1 of the month, in January"},
	{"@every 1h30m", "every 1h30m from when the schedule starts"},
	{"@triggered", "only when triggered"},
	{"CRON_TZ=UTC 0 0 * * *", "at 00:00, UTC time"},
	{"TZ=America/Los_Angeles @daily", "at 00:00, America/Los_Angeles time"},
}

func TestDescribe(t *testing.T) {
	for _, tt := range golden {
		if _, err := cron.FullParser().WithHashKey("k").Parse(tt.expr); err != nil {
			t.Errorf("%q is not valid go-cron: %v", tt.expr, err)
			continue
		}
		got, err := Describe(tt.expr)
		if err != nil || got != tt.want {
			t.Errorf("Describe(%q) = %q, %v; want %q", tt.expr, got, err, tt.want)
		}
	}
}

func TestDescribeFallsBackToLiteral(t *testing.T) {
	got, err := Describe("0 5-10/2,30 * * *")
	if err == nil || got != "minute 0, hour 5-10/2,30, day-of-month *, month *, day-of-week *" {
		t.Errorf("got %q, %v", got, err)
	}
}

// grammar is §5.1 of the design as a regular expression. Every description
// must match it, so a future parser (go-cron #299) can read them back.
func grammar() *regexp.Regexp {
	alt := func(xs ...string) string { return "(?:" + strings.Join(xs, "|") + ")" }
	opt := func(x string) string { return "(?:" + x + ")?" }
	listOf := func(w string) string {
		it := w + opt(" to "+w)
		return it + opt("(?:, "+it+")* and "+it)
	}
	num := `\d+`
	day := alt("Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday")
	month := alt("January", "February", "March", "April", "May", "June", "July",
		"August", "September", "October", "November", "December")
	sg := alt("second", "minute", "hour")
	pl := alt("seconds", "minutes", "hours")
	between := opt(" between " + num + " and " + num)
	values := alt(sg, pl) + " " + listOf(num)
	hashed := "a hashed " + sg + between
	step := alt("every "+sg, "every "+num+" "+pl) +
		opt(alt(" from "+num+" to "+num, " from a hashed start"+between))
	unit := alt(values, hashed, step)
	lead := alt("at "+values, "at "+hashed, step)
	clock := `\d\d:\d\d(?::\d\d)?`
	tm := alt("at "+clock, lead+"(?: of "+unit+")*")
	dom := alt(
		alt("day", "days")+" "+listOf(num)+" of the month",
		"the last day of the month",
		"the "+num+alt("st", "nd", "rd", "th")+" day before the end of the month",
		"the weekday nearest day "+num,
		"the last weekday of the month",
		"a hashed day of the month"+between,
		alt("every day", "every "+num+" days")+opt(" from "+num+" to "+num)+" of the month",
	)
	dow := alt(listOf(day), "the "+alt("first", "second", "third", "fourth", "fifth", "last")+" "+day,
		"a hashed day of the week")
	dayc := alt("on "+dom+opt(" when it falls on "+dow), "on "+dow)
	monthc := alt("in "+listOf(month),
		"in "+alt("every month", "every "+num+" months")+opt(" from "+month+" to "+month),
		"in a hashed month")
	year := "in " + listOf(num)
	zone := `[A-Za-z0-9_+\-/]+ time`
	sched := alt("every [0-9a-zµ.]+ from when the schedule starts", "only when triggered",
		tm+opt(", "+dayc)+opt(", "+monthc)+opt(", "+year))
	return regexp.MustCompile("^" + sched + opt(", "+zone) + "$")
}

func TestDescriptionsMatchGrammar(t *testing.T) {
	g := grammar()
	for _, tt := range golden {
		if !g.MatchString(tt.want) {
			t.Errorf("%q: description %q is outside the grammar", tt.expr, tt.want)
		}
	}
}
