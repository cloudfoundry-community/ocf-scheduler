package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/tzlist/rfc9636"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/tzlist/tzposix"
	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/spf13/pflag"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

// SchedulerJson is reserved for potential future use as an alternative to core.Timezone
// type SchedulerJson struct {
// 	Name             string   `json:"Name,omitempty"`
// 	IsServerTimeZone string   `json:"IsServerTimeZone,omitempty"`
// 	HasDst           bool     `json:"HasDst"`
// 	Std              string   `json:"Std"`
// 	Dst              string   `json:"Dst,omitempty"`
// 	Aliases          []string `json:"Aliases,omitempty"`
// 	Rules            string   `json:"Rules,omitempty"`
// }

var SemVerMajor string
var SemVerMinor string
var SemVerPatch string
var SemVerPrerelease string
var SemVerBuild string
var BuildDate string
var BuildVcsUrl string
var BuildVcsId string
var BuildVcsIdDate string

const (
	LevelTrace = slog.Level(-8)
	LevelFatal = slog.Level(12)
)

// Custom Logger methods for Trace and Fatal
func Trace(msg string, args ...any) {
	slog.Log(context.Background(), LevelTrace, msg, args...)
}

func Fatal(msg string, args ...any) {
	slog.Log(context.Background(), LevelFatal, msg, args...)
	os.Exit(1) // Terminate the program after logging
}

// Information returned by time.Zone function
type TzZoneType struct {
	Name   string
	Offset int
}

// Offsets [0] standard time
// Offsets [1] daylight savings time
// len[Offsets] > 1 Has daylight savings time

type TzInfoType struct {
	Aliases          []string
	IsServerTimeZone bool
	Offsets          []TzZoneType
	Extend           string
}

var SchedulerFilename string

type TzInfoMap map[string]TzInfoType

var TzInfos = make(TzInfoMap)

func (tzi TzInfoMap) AddZoneAlias(zone string, alias string) {
	zoneInfo, exists := tzi[zone]
	if !exists {
		zoneInfo = NewTzInfo()
	}

	tzname := time.Now().Location().String()
	if (tzname == "Local" && alias == "localtime") || tzname == alias {
		slog.Debug("IsServerTimeZone is set")
		zoneInfo.IsServerTimeZone = true
	}
	index, found := slices.BinarySearch(zoneInfo.Aliases, alias)

	if !found {
		zoneInfo.Aliases = slices.Insert(zoneInfo.Aliases, index, alias)
	}
	tzi[zone] = zoneInfo
}

func (tzi TzInfoMap) Add(zone string, data *rfc9636.Location) {
	zoneInfo, exists := tzi[zone]
	if !exists {
		zoneInfo = NewTzInfo()
	}

	localTime := time.Now()
	year := localTime.Year()
	tzname := localTime.Location().String()
	loc, err := time.LoadLocation(zone)
	if err != nil {
		Fatal("LoadLocation failed", "error", err)
	}

	if (tzname == "Local" && zone == "localtime") || tzname == zone {
		zoneInfo.IsServerTimeZone = true
		slog.Debug("IsServerTimeZone is set")
	}

	// Check offset on two dates six months apart to detect DST
	janTime := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
	julTime := time.Date(year, time.July, 1, 0, 0, 0, 0, loc)

	janName, janOffset := janTime.Zone()
	julName, julOffset := julTime.Zone()

	// Use IsDST to ensure Offsets[0] is always standard time and
	// Offsets[1] is DST, regardless of hemisphere
	if janTime.IsDST() {
		// Southern hemisphere: January is summer/DST
		zoneInfo.Offsets = append(zoneInfo.Offsets, TzZoneType{julName, julOffset})
		if janOffset != julOffset {
			zoneInfo.Offsets = append(zoneInfo.Offsets, TzZoneType{janName, janOffset})
		}
	} else {
		// Northern hemisphere or no DST: January is winter/standard
		zoneInfo.Offsets = append(zoneInfo.Offsets, TzZoneType{janName, janOffset})
		if janOffset != julOffset {
			zoneInfo.Offsets = append(zoneInfo.Offsets, TzZoneType{julName, julOffset})
		}
	}

	zoneInfo.Extend = data.Extend()
	tzi[zone] = zoneInfo
}

func NewTzInfo() TzInfoType {
	return TzInfoType{
		Aliases:          make([]string, 0),
		IsServerTimeZone: false,
		Offsets:          make([]TzZoneType, 0, 2),
		Extend:           "",
	}
}

func NewSchedulerJson(name, std, dst string, dstFlag bool, aliases []string, rules string, isServerTimeZone bool) core.Timezone {
	isSvrTz := ""
	if isServerTimeZone {
		isSvrTz = "yes"
	}
	return core.Timezone{
		Name:             name,
		IsServerTimeZone: isSvrTz,
		HasDst:           dstFlag,
		Std:              std,
		Dst:              dst,
		Aliases:          aliases,
		Rules:            rules,
	}
}

func GenerateJson(zones []string) {
	schedulerZones := make([]core.Timezone, 0, len(zones))
	for _, name := range zones {
		if zone, exist := TzInfos[name]; exist {
			std, dst, rules, err := tzposix.DecodeTZ(zone.Extend)
			if err != nil {
				slog.Error("DecodeTZ failure", "TZ", zone.Extend, "error", err)
			}
			zj := NewSchedulerJson(name, std, dst, len(zone.Offsets) > 1, zone.Aliases, rules, zone.IsServerTimeZone)
			schedulerZones = append(schedulerZones, zj)
		} else {
			slog.Warn("Missing zone", "name", name)
		}
	}

	jsonData, err := json.MarshalIndent(schedulerZones, "", "  ")
	if err != nil {
		Fatal("Error marshaling to JSON ", "error", err)
	}

	err = os.WriteFile(SchedulerFilename, jsonData, 0644)
	if err != nil {
		Fatal("Error writing to file", "error", err)
	}
	slog.Info("Successfully wrote JSON data", "file", SchedulerFilename)
}

func main() {
	bm := createBuildMeta(SemVerBuild)
	sv := createSemVer(SemVerMajor, SemVerMinor, SemVerPatch, SemVerPrerelease, bm)
	slog.Info("tzlist", "version", sv, "build date", BuildDate)
	slog.Info("Vcs Info", "url", BuildVcsUrl, "id", BuildVcsId, "date", BuildVcsIdDate)
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "NAME:\n   %s - Generate scheduler's timezones file\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "USAGE:\n   tzlist [OPTIONS]\n\n")

		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		pflag.PrintDefaults()
	}
	logLevels := map[string]slog.Level{
		"t": LevelTrace, "trace": LevelTrace,
		"d": slog.LevelDebug, "debug": slog.LevelDebug,
		"i": slog.LevelInfo, "info": slog.LevelInfo,
		"w": slog.LevelWarn, "warn": slog.LevelWarn, "warning": slog.LevelWarn,
		"e": slog.LevelError, "error": slog.LevelError,
		"f": LevelFatal, "fatal": LevelFatal,
	}
	pflag.FuncP("loglevel", "l", "Set loglevel to trace, debug, info, warning, error or fatal", func(value string) error {
		if level, ok := logLevels[strings.ToLower(value)]; ok {
			slog.SetLogLoggerLevel(level)
			return nil
		}
		return errors.New("The loglevel parameter value must be one of: trace (t), debug (d), info (i), warning (w), error (e), or fatal (f).")
	})
	pflag.StringVarP(&SchedulerFilename, "json", "j", "", "pathname to write the json file to")
	jsonPath := filepath.Join(core.TimezoneJsonDir, core.TimezoneJsonBase)
	pflag.Lookup("json").NoOptDefVal = jsonPath
	// Parsed Arguments	Resulting Value
	// --json=hulu		hulu
	// --json		scheduler.json
	// [nothing]		""

	pflag.Parse()

	numAliases := 0
	zones, keylen := GetOsTimeZones()
	if len(SchedulerFilename) > 0 {
		GenerateJson(zones)
		return
	}

	keylen += 4 // for output spacing
	slog.Info("Statistics", "numKeys", len(zones), "keylen", keylen)
	for _, name := range zones {
		zone, exist := TzInfos[name]
		if exist {
			var description string
			var err error

			description, err = tzposix.HumanReadableTZ(zone.Extend)
			if err != nil {
				slog.Error("HumanReadableTZ failure", "extend", zone.Extend, "error", err)
			}
			if zone.IsServerTimeZone {
				name = name + "*"
			}

			dstLabel := "no"
			if len(zone.Offsets) > 1 {
				dstLabel = "yes"
			}
			fmt.Printf("%-*s DST: %-3s %+v Extend %s\n", keylen, name, dstLabel, zone.Aliases, zone.Extend)
			if len(description) != 0 {
				fmt.Println(description)
			}
		} else {
			slog.Warn("Missing zone", "name", name)
			continue
		}
		numAliases += len(zone.Aliases)
	}
	slog.Info("Statistics", "zoneinfos", len(zones), "aliases", numAliases, "total", len(zones)+numAliases)
}

func GetOsTimeZones() ([]string, int) {
	var zoneDirs = []string{
		// Update path according to your OS
		"/usr/share/zoneinfo",
		"/usr/share/lib/zoneinfo",
		"/usr/lib/locale/TZ",
	}

	for _, zd := range zoneDirs {
		walkTzDir(zd, zd)
	}

	zones := make([]string, 0, len(TzInfos))
	keylen := 0
	for key := range TzInfos {
		zones = append(zones, key)
		if len(key) > keylen {
			keylen = len(key)
		}
	}
	sort.Strings(zones)

	return zones, keylen
}

func walkTzDir(root, path string) {
	dirInfos, err := os.ReadDir(path)
	if err != nil {
		Trace("zoneinfo directory is not available", "path", path)
		return
	}

	// Linux Convention
	//   The zoneinfo names are capitalized.  We can ignore directories and files that do not follow
	//   that convention for now. We might need an exception list in the future if we want to allow
	//   localtime and posixrules zoneinfo files to be processed as well

	for _, info := range dirInfos {
		if info.IsDir() && info.Name() != strings.ToUpper(info.Name()[:1])+info.Name()[1:] {
			Trace("Skipping directory because name is not capitalized ", "filename", info.Name())
			continue
		}

		newPath := filepath.Join(path, info.Name())

		if info.IsDir() {
			walkTzDir(root, newPath)
		} else {
			relPath, err := filepath.Rel(root, newPath)
			if err != nil {
				slog.Error("Could not determine relative path", "root", root, "path", newPath, "error", err)
				continue
			}
			// Use os.Lstat to reliably detect symlinks; os.ReadDir's DirEntry.Type()
			// follows symlinks on most systems, so ModeSymlink is never set there.
			fileInfo, lstatErr := os.Lstat(newPath)
			if lstatErr != nil {
				slog.Error("Could not lstat file", "path", newPath, "error", lstatErr)
				continue
			}
			isSymlink := fileInfo.Mode()&os.ModeSymlink != 0

			if zoneInfo, err := rfc9636.LoadLocation(relPath, []string{root}); err == nil {
				slog.Debug("dump of zoneinfo", "timezone", relPath)
				if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
					rfc9636.DumpLocation(zoneInfo)
				}

				if isSymlink {
					symTarget, err := os.Readlink(newPath)
					if err != nil {
						slog.Error("Could not read link target", "path", newPath, "error", err)
						continue
					}
					slog.Debug("source points to target", "source", newPath, "target", symTarget)
					resolvedPath, err := filepath.EvalSymlinks(newPath)
					if err != nil {
						slog.Error("Could not evaluate symlink", "symlink", newPath, "error", err)
						continue
					}
					atz, found := strings.CutPrefix(resolvedPath, root+"/")
					if !found {
						slog.Error("Could not extract timezone alias", "path", resolvedPath)
						continue
					}
					slog.Debug("Timezone has alias", "timezone", atz, "alias", relPath)
					TzInfos.AddZoneAlias(atz, relPath)
				} else {
					TzInfos.Add(relPath, zoneInfo)
				}

			} else {
				Trace("File is not a timezone file", "file", newPath)
			}

		}
	}
}
