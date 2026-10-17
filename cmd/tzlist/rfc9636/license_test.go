package rfc9636

import (
	"strings"
	"testing"
)

// The BSD-3 binary clause needs the copyright notice, all three
// conditions and the disclaimer reproduced, so a truncated LICENSE
// must fail here rather than ship.
func TestLicenseCarriesNoticeConditionsAndDisclaimer(t *testing.T) {
	for _, want := range []string{
		"Copyright 2009 The Go Authors.",
		"Redistributions of source code must retain",
		"Redistributions in binary form must reproduce",
		"Neither the name of Google LLC",
		"THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS",
		"EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.",
	} {
		if !strings.Contains(License, want) {
			t.Errorf("License is missing %q", want)
		}
	}
}
