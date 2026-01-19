package tzposix

import (
	_ "fmt"
	"strings"
	"testing"
)

func TestHumanReadableTZAll(t *testing.T) {
	var tests = []string{
		"<+00>0<+01>,0/0,J365/25",
		"<+00>0<+02>-2,M3.5.0/1,M10.5.0/3",
		"<+01>-1",
		"<+02>-2",
		"<+0330>-3:30",
		"<+03>-3",
		"<+0430>-4:30",
		"<+04>-4",
		"<+0530>-5:30",
		"<+0545>-5:45",
		"<+05>-5",
		"<+0630>-6:30",
		"<+06>-6",
		"<+07>-7",
		"<+0845>-8:45",
		"<+0845>-8:45:15",
		"<+0845>-8:45:59",
		"<+08>-8",
		"<+09>-9",
		"<+1030>-10:30<+11>-11,M10.1.0,M4.1.0",
		"<+10>-10",
		"<+11>-11",
		"<+11>-11<+12>,M10.1.0,M4.1.0/3",
		"<+1245>-12:45<+1345>,M9.5.0/2:45,M4.1.0/3:45",
		"<+12>-12",
		"<+13>-13",
		"<+14>-14",
		"<-00>0",
		"<-01>1",
		"<-01>1<+00>,M3.5.0/0,M10.5.0/1",
		"<-02>2",
		"<-02>2<-01>,M3.5.0/-1,M10.5.0/0",
		"<-03>3",
		"<-03>3<-02>,M3.2.0,M11.1.0",
		"<-04>4",
		"<-04>4<-03>,M9.1.6/24,M4.1.6/24",
		"<-05>5",
		"<-06>6",
		"<-06>6<-05>,M9.1.6/22,M4.1.6/22",
		"<-07>7",
		"<-08>8",
		"<-0930>9:30",
		"<-09>9",
		"<-10>10",
		"<-11>11",
		"<-12>12",
		"ACST-9:30",
		"ACST-9:30ACDT,M10.1.0,M4.1.0/3",
		"AEST-10",
		"AEST-10AEDT,M10.1.0,M4.1.0/3",
		"AKST9AKDT,M3.2.0,M11.1.0",
		"AST4",
		"AST4ADT,M3.2.0,M11.1.0",
		"AWST-8",
		"CAT-2",
		"CET-1",
		"CET-1CEST,M3.5.0,M10.5.0/3",
		"CST-8",
		"CST5CDT,M3.2.0/0,M11.1.0/1",
		"CST6",
		"CST6CDT,M3.2.0,M11.1.0",
		"ChST-10",
		"EAT-3",
		"EET-2",
		"EET-2EEST,M3.4.4/50,M10.4.4/50",
		"EET-2EEST,M3.5.0,M10.5.0/3",
		"EET-2EEST,M3.5.0/0,M10.5.0/0",
		"EET-2EEST,M3.5.0/3,M10.5.0/4",
		"EET-2EEST,M4.5.5/0,M10.5.4/24",
		"EST5",
		"EST5EDT,M3.2.0,M11.1.0",
		"GMT0",
		"GMT0BST,M3.5.0/1,M10.5.0",
		"GMT0IST,M3.5.0/1,M10.5.0",
		"HKT-8",
		"HST10",
		"HST10HDT,M3.2.0,M11.1.0",
		"IST-2IDT,M3.4.4/26,M10.5.0",
		"IST-5:30",
		"JST-9",
		"KST-9",
		"MET-1MEST,M3.5.0,M10.5.0/3",
		"MSK-3",
		"MST7",
		"MST7MDT,M3.2.0,M11.1.0",
		"NST3:30NDT,M3.2.0,M11.1.0",
		"NZST-12NZDT,M9.5.0,M4.1.0/3",
		"PKT-5",
		"PST-8",
		"PST8PDT,M3.2.0,M11.1.0",
		"SAST-2",
		"SST11",
		"UTC0",
		"WAT-1",
		"WET0WEST,M3.5.0/1,M10.5.0",
		"WIB-7",
		"WIT-9",
		"WITA-8",
	}

	for _, ptz := range tests {

		t.Run(ptz, func(t *testing.T) {
			t.Log(ptz)
			ans, err := HumanReadableTZ(ptz)
			if err != nil {
				t.Errorf("got %v, want nil", err)
				return
			}
			t.Log(ans)
		})
	}
}

/*
TODO Add these tests
CET6CEST530,M4.5.0/02:00:00,M10.5.0/03:00:00
HAST10HADT,M4.2.0/03:0:0,M10.2.0/03:0:00
AST9ADT,M3.2.0,M11.1.0
AST9ADT,M3.2.0/03:0:0,M11.1.0/03:0:0
EST5EDT,M3.2.0/02:00:00,M11.1.0/02:00:00
America/Sao_Paulo   GRNLNDST3GRNLNDDT,M10.3.0/00:00:00,M2.4.0/00:00:00
EST5EDT,M3.2.0/02:00:00,M11.1.0
EST5EDT,M3.2.0,M11.1.0/02:00:00
CST6CDT,M3.2.0/2:00:00,M11.1.0/2:00:00
MST7MDT,M3.2.0/2:00:00,M11.1.0/2:00:00
PST8PDT,M3.2.0/2:00:00,M11.1.0/2:00:00
*/

func TestHumanReadableTZ(t *testing.T) {
	var tests = []struct {
		tz          string
		expectSst   string
		expectDst   string
		expectRules string
		expectError error
	}{
		{tz: "IST-2IDT,M3.4.4/26,M10.5.0",
			expectSst:   "Standard Time: IST (UTC +02:00)",
			expectDst:   "Daylight Time: IDT (UTC +03:00)",
			expectRules: "Rules: Starts on the Friday on or after March 23rd at 02:00:00, Ends on the last Sunday of October at 02:00:00",
			expectError: nil,
		},
		{tz: "EET-2EEST,M3.4.4/50,M10.4.4/50",
			expectSst:   "Standard Time: EET (UTC +02:00)",
			expectDst:   "Daylight Time: EEST (UTC +03:00)",
			expectRules: "Rules: Starts on the fourth Thursday of March at 00:50:00, Ends on the fourth Thursday of October at 00:50:00",
			expectError: nil,
		},
		{tz: "EET-2EEST,M4.5.5/0,M10.5.4/24",
			expectSst:   "Standard Time: EET (UTC +02:00)",
			expectDst:   "Daylight Time: EEST (UTC +03:00)",
			expectRules: "Rules: Starts on the last Friday of April at 00:00:00, Ends on the last Thursday of October at midnight of the next day",
			expectError: nil,
		},
		{tz: "PST8PDT,M3.2.0,M11.1.0",
			expectSst:   "Standard Time: PST (UTC -08:00)",
			expectDst:   "Daylight Time: PDT (UTC -07:00)",
			expectRules: "Rules: Starts on the second Sunday of March at 02:00:00, Ends on the first Sunday of November at 02:00:00",
			expectError: nil,
		},
		{tz: "NST-3:30NDT2:30,M3.2.0/2:30:2,M11.1.0/11:25:40",
			expectSst:   "Standard Time: NST (UTC +03:30)",
			expectDst:   "Daylight Time: NDT (UTC -02:30)",
			expectRules: "Rules: Starts on the second Sunday of March at 02:30:02, Ends on the first Sunday of November at 11:25:40",
			expectError: nil,
		},
		{tz: "CST6CDT,M3.2.0/2:00:00,M11.1.0/2:00:00",
			expectSst:   "Standard Time: CST (UTC -06:00)",
			expectDst:   "Daylight Time: CDT (UTC -05:00)",
			expectRules: "Rules: Starts on the second Sunday of March at 02:00:00, Ends on the first Sunday of November at 02:00:00",
			expectError: nil,
		},
		{tz: "GRNLNDST3GRNLNDDT,M10.3.0/00:00:00,M2.4.0/00:00:00",
			expectSst:   "Standard Time: GRNLNDST (UTC -03:00)",
			expectDst:   "Daylight Time: GRNLNDDT (UTC -02:00)",
			expectRules: "Rules: Starts on the third Sunday of October at 00:00:00, Ends on the fourth Sunday of February at 00:00:00",
			expectError: nil,
		},
		{tz: "<+00>0<+01>,0/0,J365/25",
			expectSst:   "Standard Time: <+00> (UTC +00:00)",
			expectDst:   "Daylight Time: <+01> (UTC +01:00)",
			expectRules: "Rules: Starts from the start of the year, Ends at the end of the year",
			expectError: nil,
		},
	}

	for _, tt := range tests {

		testname := tt.tz
		t.Run(testname, func(t *testing.T) {
			ans, err := HumanReadableTZ(tt.tz)
			if err != nil {
				t.Errorf("got %v, want nil", err)
				return
			}
			parts := strings.Split(ans, "\n")
			if len(parts) != 3 {
				t.Errorf("got %d parts, want 3\nactual %s", len(parts), ans)
				return
			}
			if parts[0] != tt.expectSst {
				t.Errorf("got %s want %s\nactual %s", parts[0], tt.expectSst, ans)
				return
			}
			if parts[1] != tt.expectDst {
				t.Errorf("got %s want %s\nactual %s", parts[1], tt.expectDst, ans)
				return
			}
			if parts[2] != tt.expectRules {
				t.Errorf("got %s want %s\nactual %s", parts[2], tt.expectRules, ans)
				return
			}
		})
	}
}

func TestHumanReadableTZNoDst(t *testing.T) {
	var tests = []struct {
		tz          string
		expectSst   string
		expectDst   string
		expectRules string
		expectError error
	}{
		{tz: "GMT0",
			expectSst:   "Standard Time: GMT (UTC +00:00)",
			expectRules: "(No Daylight Saving Time rules)",
			expectError: nil,
		},
		{tz: "EAT-3",
			expectSst:   "Standard Time: EAT (UTC +03:00)",
			expectRules: "(No Daylight Saving Time rules)",
			expectError: nil,
		},
	}

	for _, tt := range tests {

		testname := tt.tz
		t.Run(testname, func(t *testing.T) {
			ans, err := HumanReadableTZ(tt.tz)
			if err != nil {
				t.Errorf("got %v, want nil", err)
				return
			}
			parts := strings.Split(ans, "\n")
			if len(parts) != 2 {
				t.Errorf("got %d parts, want 2", len(parts))
				return
			}
			if parts[0] != tt.expectSst {
				t.Errorf("got %s parts, want %s", parts[0], tt.expectSst)
				return
			}
			if parts[1] != tt.expectRules {
				t.Errorf("got %s parts, want %s", parts[1], tt.expectRules)
				return
			}
		})
	}
}
