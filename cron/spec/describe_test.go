package spec

import (
	"regexp"
	"strings"
	"testing"

	cron "github.com/netresearch/go-cron"
)

var golden = []struct{ expr, want string }{
	{"*/15 * * * *", "at minutes 0, 15, 30 and 45 of every hour"},
	{"* * * * *", "every minute"},
	{"* * * * * *", "every second"},
	{"*/10 * * * * *", "at seconds 0, 10, 20, 30, 40 and 50 of every minute"},
	{"*/2 * * * * *", "at seconds 0, 2, 4, … 58 of every minute (every 2 seconds)"},
	{"0 30 2 * * MON-FRI", "at 02:30, on Monday to Friday"},
	{"15 30 2 * * *", "at 02:30:15"},
	{"0 9-17 * * MON-FRI", "at minute 0 of hours 9 to 17, on Monday to Friday"},
	{"5,10,40 * * * *", "at minutes 5, 10 and 40 of every hour"},
	{"* 2 * * *", "every minute of hour 2"},
	{"30 */2 * * *", "at minute 30 of hours 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20 and 22 of every day"},
	{"0 */2 * * MON", "at minute 0 of hours 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20 and 22, on Monday"},
	{"0 0-21/3 * * *", "at minute 0 of hours 0, 3, 6, 9, 12, 15, 18 and 21 of every day"},
	{"*/15 9-17 * * *", "at minutes 0, 15, 30 and 45 of hours 9 to 17"},
	{"0 22-2 * * *", "at 22:00, 23:00, 00:00, 01:00 and 02:00"},
	{"5-50/15 * * * *", "at minutes 5, 20, 35 and 50 of every hour"},
	{"5/15 * * * *", "at minutes 5, 20, 35 and 50 of every hour"},
	{"CRON_TZ=US/Pacific 5/15 * * * *", "at minutes 5, 20, 35 and 50 of every hour, US/Pacific time"},
	{"10-40/10 * * * *", "at minutes 10, 20, 30 and 40 of every hour"},
	{"*/7 * * * *", "at minutes 0, 7, 14, … 56 of every hour (every 7 minutes, starting over each hour)"},
	{"*/7 9-10 * * *", "at minutes 0, 7, 14, … 56 of hours 9 to 10 (every 7 minutes, starting over each hour)"},
	{"5/15 2 * * *", "at 02:05, 02:20, 02:35 and 02:50"},
	{"*/15 2 * * *", "at 02:00, 02:15, 02:30 and 02:45"},
	{"*/20 30 * * * *", "at seconds 0, 20 and 40 of minute 30 of every hour"},
	{"15 */4 * * * *", "at second 15 of minutes 0, 4, 8, … 56 of every hour (every 4 minutes)"},
	{"0 1/6 * * *", "at 01:00, 07:00, 13:00 and 19:00"},
	{"30 */6 * * *", "at 00:30, 06:30, 12:30 and 18:30"},
	{"0 9,17 * * *", "at 09:00 and 17:00"},
	{"15 0 1/6 * * *", "at 01:00:15, 07:00:15, 13:00:15 and 19:00:15"},
	{"H(15-45) 2 * * *", "at a hashed minute between 15 and 45 of hour 2"},
	{"H H * * *", "at a hashed minute of a hashed hour"},
	{"H/15 * * * *", "every 15 minutes from a hashed start"},
	{"H(15-45)/10 2 * * *", "every 10 minutes from a hashed start between 15 and 45 of hour 2"},
	{"0 0 13 * FRI", "at 00:00, on day 13 of the month when it falls on Friday"},
	{"0 0 1,15 * *", "at 00:00, on days 1 and 15 of the month"},
	{"0 0 1/10 * *", "at 00:00, on days 1, 11, 21 and 31 of every month"},
	{"0 0 5/10 * *", "at 00:00, on days 5, 15 and 25 of every month"},
	{"0 0 5-25/10 * *", "at 00:00, on days 5, 15 and 25 of every month"},
	{"0 0 1/2 * *", "at 00:00, on days 1, 3, 5, … 31 of every month (every 2 days, starting over each month)"},
	{"0 0 L * *", "at 00:00, on the last day of the month"},
	{"0 0 L-3 * *", "at 00:00, on the 3rd day before the end of the month"},
	{"0 0 LW * *", "at 00:00, on the last weekday of the month"},
	{"0 0 15W * *", "at 00:00, on the weekday nearest day 15"},
	{"0 0 * * FRI#3", "at 00:00, on the third Friday"},
	{"0 0 * * 5#L", "at 00:00, on the last Friday"},
	{"0 0 * * 0,7", "at 00:00, on Sunday and Sunday"},
	{"0 0 * * 1/2", "at 00:00, on Monday, Wednesday, Friday and Sunday"},
	{"0 0 * * */2", "at 00:00, on Sunday, Tuesday, Thursday and Saturday"},
	{"0 0 * * 0/1", "at 00:00, on Sunday, Monday, Tuesday, Wednesday, Thursday, Friday and Saturday"},
	{"0 0 1 1 * 2027", "at 00:00, on day 1 of the month, in January, in 2027"},
	{"0 0 1 */3 *", "at 00:00, on day 1 of the month, in January, April, July and October"},
	{"0 0 1 2/3 *", "at 00:00, on day 1 of the month, in February, May, August and November"},
	{"0 0 1 2-8/3 *", "at 00:00, on day 1 of the month, in February, May and August"},
	{"0 0 1 JAN-MAR *", "at 00:00, on day 1 of the month, in January to March"},
	{"0 30 2 * 3,11 SUN#1,SUN#2", "at 02:30, on the first Sunday and the second Sunday, in March and November"},
	{"0 9 * * MON,FRI#L", "at 09:00, on Monday and the last Friday"},
	{"0 9 * * MON-WED,FRI#3", "at 09:00, on Monday to Wednesday and the third Friday"},
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

// A hashed step's description stays short ("every 15 minutes from a hashed
// start"); its note spells out the values in terms of the hashed start.
var noteGolden = []struct{ expr, want string }{
	{"H/15 * * * *", "at minutes h, h+15, h+30 and h+45 of every hour, where h is a hashed minute from 0 to 14"},
	{"H/5 * * * *", "at minutes h, h+5, h+10, … h+55 of every hour, where h is a hashed minute from 0 to 4 (every 5 minutes)"},
	{"H/7 * * * *", "at minutes h, h+7, h+14, … up to 59 of every hour, where h is a hashed minute from 0 to 6 (every 7 minutes, starting over each hour)"},
	{"H(15-45)/10 2 * * MON", "at minutes h, h+10, h+20, … up to 45 of hour 2, where h is a hashed minute from 15 to 24 (every 10 minutes, starting over each hour)"},
	{"H(10-20)/30 * * * *", "at minute h of every hour, where h is a hashed minute from 10 to 20"},
	{"0 H/6 * * *", "at minute 0 of hours h, h+6, h+12 and h+18 of every day, where h is a hashed hour from 0 to 5"},
	{"H/30 H/12 * * *", "at minutes h and h+30 of hours k and k+12 of every day, where h is a hashed minute from 0 to 29 and k is a hashed hour from 0 to 11"},
	{"CRON_TZ=UTC H/15 * * * *", "at minutes h, h+15, h+30 and h+45 of every hour, where h is a hashed minute from 0 to 14"},
	{"*/15 * * * *", ""},
	{"H 2 * * *", ""},
	{"@daily", ""},
	{"0 5-10/2,22 * * *", ""},
}

func TestDescribeNote(t *testing.T) {
	for _, tt := range noteGolden {
		if _, err := cron.FullParser().WithHashKey("k").Parse(tt.expr); err != nil {
			t.Errorf("%q is not valid go-cron: %v", tt.expr, err)
			continue
		}
		if got := DescribeNote(tt.expr); got != tt.want {
			t.Errorf("DescribeNote(%q) = %q; want %q", tt.expr, got, tt.want)
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
	// a value list, or one cut short: "0, 7, 14, … 56"
	vlist := func(w string) string { return alt(listOf(w), w+", "+w+", "+w+", … "+w) }
	num := `\d+`
	day := alt("Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday")
	month := alt("January", "February", "March", "April", "May", "June", "July",
		"August", "September", "October", "November", "December")
	sg := alt("second", "minute", "hour")
	pl := alt("seconds", "minutes", "hours")
	between := opt(" between " + num + " and " + num)
	interval := "every " + alt(sg, "day", num+" "+alt(pl, "days")) +
		opt(", starting over each "+alt("minute", "hour", "day", "month"))
	note := " \\(" + interval + "(?:; " + interval + ")*\\)"
	values := alt(sg, pl) + " " + vlist(num)
	hashed := "a hashed " + sg + between
	hstep := "every " + num + " " + pl + " from a hashed start" + between
	unit := alt(values, hashed, hstep, "every "+sg, "every day")
	lead := alt("at "+values, "at "+hashed, hstep, "every "+sg)
	clock := `\d\d:\d\d(?::\d\d)?`
	clocks := clock + opt("(?:, "+clock+")* and "+clock)
	tm := alt("at "+clocks, lead+"(?: of "+unit+")*"+opt(note))
	dom := alt(
		alt("day", "days")+" "+listOf(num)+" of the month",
		"the last day of the month",
		"the "+num+alt("st", "nd", "rd", "th")+" day before the end of the month",
		"the weekday nearest day "+num,
		"the last weekday of the month",
		"a hashed day of the month"+between,
		alt("day", "days")+" "+vlist(num)+" of every month"+opt(note),
	)
	dowItem := alt(day+opt(" to "+day), "the "+alt("first", "second", "third", "fourth", "fifth", "last")+" "+day)
	dow := alt(dowItem+opt("(?:, "+dowItem+")* and "+dowItem), "a hashed day of the week")
	dayc := alt("on "+dom+opt(" when it falls on "+dow), "on "+dow)
	monthc := alt("in "+listOf(month), "in a hashed month")
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
