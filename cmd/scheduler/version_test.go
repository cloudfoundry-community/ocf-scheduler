package main

import (
	"runtime"
	"testing"
)

func TestCreateBuildMeta(t *testing.T) {
	var tests = []struct {
		name   string
		build  string
		expect string
	}{
		{
			name:   "empty build",
			build:  "",
			expect: runtime.GOOS + "." + runtime.GOARCH,
		},
		{
			name:   "with build string",
			build:  "abc123",
			expect: runtime.GOOS + "." + runtime.GOARCH + ".abc123",
		},
		{
			name:   "whitespace-only build",
			build:  "   ",
			expect: runtime.GOOS + "." + runtime.GOARCH,
		},
		{
			name:   "build with whitespace",
			build:  "  abc123  ",
			expect: runtime.GOOS + "." + runtime.GOARCH + ".abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createBuildMeta(tt.build)
			if result != tt.expect {
				t.Errorf("got %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestCreateSemVer(t *testing.T) {
	var tests = []struct {
		name       string
		major      string
		minor      string
		patch      string
		prerelease string
		build      string
		expect     string
	}{
		{
			name:  "basic version",
			major: "1", minor: "2", patch: "3",
			expect: "1.2.3",
		},
		{
			name:  "with prerelease",
			major: "1", minor: "0", patch: "0",
			prerelease: "beta",
			expect:     "1.0.0-beta",
		},
		{
			name:  "with build",
			major: "1", minor: "0", patch: "0",
			build:  "darwin.arm64",
			expect: "1.0.0+darwin.arm64",
		},
		{
			name:  "with prerelease and build",
			major: "2", minor: "1", patch: "0",
			prerelease: "rc1",
			build:      "linux.amd64",
			expect:     "2.1.0-rc1+linux.amd64",
		},
		{
			name:  "with whitespace",
			major: " 1 ", minor: " 2 ", patch: " 3 ",
			expect: "1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createSemVer(tt.major, tt.minor, tt.patch, tt.prerelease, tt.build)
			if result != tt.expect {
				t.Errorf("got %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestCreateSemVerPanicsOnMissingParts(t *testing.T) {
	var tests = []struct {
		name  string
		major string
		minor string
		patch string
	}{
		{name: "missing major", major: "", minor: "1", patch: "0"},
		{name: "missing minor", major: "1", minor: "", patch: "0"},
		{name: "missing patch", major: "1", minor: "0", patch: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic, got none")
				}
			}()
			createSemVer(tt.major, tt.minor, tt.patch, "", "")
		})
	}
}
