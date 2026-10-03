package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// HasDst needs both signals: Jan/Jul offsets differ this year AND the
// footer (ongoing rule) has DST. A zone that abolished DST mid-year
// still shows two offsets that year, but its footer has no DST.
func TestGenerateJsonHasDstNeedsOffsetsAndFooter(t *testing.T) {
	TzInfos = TzInfoMap{
		"America/Los_Angeles": {OffsetsDiffer: true, Extend: "PST8PDT,M3.2.0,M11.1.0"},
		"America/Vancouver":   {OffsetsDiffer: true, Extend: "MST7"},
		"Etc/RuleOnly":        {OffsetsDiffer: false, Extend: "PST8PDT,M3.2.0,M11.1.0"},
	}
	SchedulerFilename = filepath.Join(t.TempDir(), "tz.json")

	GenerateJson([]string{"America/Los_Angeles", "America/Vancouver", "Etc/RuleOnly"})

	data, err := os.ReadFile(SchedulerFilename)
	if err != nil {
		t.Fatal(err)
	}
	var zones []core.Timezone
	if err := json.Unmarshal(data, &zones); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"America/Los_Angeles": true, "America/Vancouver": false, "Etc/RuleOnly": false}
	for _, z := range zones {
		if z.HasDst != want[z.Name] {
			t.Errorf("%s: HasDst = %v, want %v", z.Name, z.HasDst, want[z.Name])
		}
	}
}
