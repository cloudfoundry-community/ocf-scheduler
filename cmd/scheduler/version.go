package main

import (
	"strings"
	"fmt"
	"runtime"
)

func createBuildMeta(build string) string {
	p1 := runtime.GOOS
	p2 := runtime.GOARCH
	p3 := strings.TrimSpace(build)
	if p1 == "" || p2 == "" {
		panic(fmt.Sprintf("Go meta data is missing one of its parts: %s, %s ", p1, p2))
	}
	b := strings.Join([]string{p1, p2}, ".")
	if p3 != "" {
		b += "." + p3
	}
	return b
}

func createSemVer(major, minor, patch, prerelease, build string) string {
	p1 := strings.TrimSpace(major)
	p2 := strings.TrimSpace(minor)
	p3 := strings.TrimSpace(patch)
	p4 := strings.TrimSpace(prerelease)
	p5 := strings.TrimSpace(build)
	if p1 == "" || p2 == "" || p3 == "" {
		panic(fmt.Sprintf("Semanic version is missing one of its parts: %s.%s.%s", p1, p2, p3))
	}

	sv := strings.Join([]string{p1, p2, p3}, ".")
	if p4 != "" {
		sv += "-" + p4
	}
	if p5 != "" {
		sv += "+" + p5
	}
	return sv
}
