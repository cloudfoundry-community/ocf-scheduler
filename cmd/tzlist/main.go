package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/tzlist/tzposix"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/tzlist/rfc9636"
	"io/ioutil"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

type SchedulerJson struct {
	Name    string   `json:"Name,omitempty"`
	HasDst  bool     `json:"HasDst"`
	Std     string   `json:"Std"`
	Dst     string   `json:"Dst,omitempty"`
	Aliases []string `json:"Aliases,omitempty"`
	Rules   string   `json:"Rules,omitempty"`
}

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
	Aliases []string
	Offsets []TzZoneType
	Extend  string
}

var SchedulerZoneSlices []SchedulerJson = make([]SchedulerJson, 0, 800)

type SchedulerZoneMap map[string]SchedulerJson

var SchedulerZoneObjects = make(SchedulerZoneMap)

var SchedulerFilename string

type TzInfoMap map[string]TzInfoType

var jsonFileFormat string = "slices"

var TzInfos = make(TzInfoMap)

func (tzi TzInfoMap) AddZoneAlias(zone string, alias string) {
	zoneInfo, exists := tzi[zone]
	if !exists {
		zoneInfo = NewTzInfo()
	}
	index, found := slices.BinarySearch(zoneInfo.Aliases, alias)

	if !found {
		zoneInfo.Aliases = slices.Insert(zoneInfo.Aliases, index, alias)
	}
	tzi[zone] = zoneInfo
	return

}

func (tzi TzInfoMap) Add(zone string, data *rfc9636.Location) {
	zoneInfo, exists := tzi[zone]
	if !exists {
		zoneInfo = NewTzInfo()
	}

	year := time.Now().Year()
	loc, err := time.LoadLocation(zone)
	if err != nil {
		Fatal("LoadLocation failed", "error", err)

	}

	// Check offset on a winter date (Jan 1) and a summer date (Jul 1)
	winterTime := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
	summerTime := time.Date(year, time.July, 1, 0, 0, 0, 0, loc)

	xst, winterOffset := winterTime.Zone()
	xdt, summerOffset := summerTime.Zone()

	zoneInfo.Offsets = append(zoneInfo.Offsets, TzZoneType{xst, winterOffset})

	if winterOffset != summerOffset {
		zoneInfo.Offsets = append(zoneInfo.Offsets, TzZoneType{xdt, summerOffset})
	}

	zoneInfo.Extend = data.Extend()
	tzi[zone] = zoneInfo
}

func NewTzInfo() TzInfoType {
	return TzInfoType{
		Aliases: make([]string, 0),
		Offsets: make([]TzZoneType, 0, 2),
		Extend:  "",
	}
}

func SupportsDST(numOffsets int) string {
	if numOffsets == 2 {
		return "yes"
	}
	return "no"
}

func NewSchedulerJson(name, std, dst string, dstFlag bool, aliases []string, rules string) SchedulerJson {
	return SchedulerJson{
		Name:    name,
		HasDst:  dstFlag,
		Std:     std,
		Dst:     dst,
		Aliases: aliases,
		Rules:   rules,
	}
}

func GenerateJson(zones []string) {
	for _, name := range zones {
		if zone, exist := TzInfos[name]; exist {
			std, dst, rules, err := tzposix.DecodeTZ(zone.Extend)
			if err != nil {
				slog.Error("DecodeTZ failure", "TZ", zone.Extend, "error", err)
			}
			if jsonFileFormat == "slices" {
				zj := NewSchedulerJson(name, std, dst, len(zone.Offsets) > 1, zone.Aliases, rules)
				SchedulerZoneSlices = append(SchedulerZoneSlices, zj)
			} else if jsonFileFormat == "objects" {
				zj := NewSchedulerJson("", std, dst, len(zone.Offsets) > 1, zone.Aliases, rules)
				SchedulerZoneObjects[name] = zj
			}

		} else {
			fmt.Printf("Missing zone %s\n", name)
		}
	}
	var jsonData []byte
	var err error
	if jsonFileFormat == "slices" {
		jsonData, err = json.MarshalIndent(SchedulerZoneSlices, "", "  ") // Use MarshalIndent for pretty print
		if err != nil {
			Fatal("Error marshaling to JSON ", "error", err)
		}
	} else if jsonFileFormat == "objects" {
		jsonData, err = json.MarshalIndent(SchedulerZoneObjects, "", "  ") // Use MarshalIndent for pretty print
		if err != nil {
			Fatal("Error marshaling to JSON ", "error", err)
		}
	}

	// 4. Write the JSON data to a file
	err = ioutil.WriteFile(SchedulerFilename, jsonData, 0644)
	if err != nil {
		Fatal("Error writing to file", "error", err)
	}
	fmt.Printf("Successfully wrote JSON data to %s\n", SchedulerFilename)
}

func main() {
	pflag.FuncP("loglevel", "l", "Set loglevel to trace, debug, info, warning, error or fatal", func(value string) error {
		lv := strings.ToLower(value)
		if strings.HasPrefix("trace", lv) {
			slog.SetLogLoggerLevel(LevelTrace)
		} else if strings.HasPrefix("debug", lv) {
			slog.SetLogLoggerLevel(slog.LevelDebug)
		} else if strings.HasPrefix("info", lv) {
			slog.SetLogLoggerLevel(slog.LevelInfo)
		} else if strings.HasPrefix("warning", lv) {
			slog.SetLogLoggerLevel(slog.LevelWarn)
		} else if strings.HasPrefix("error", lv) {
			slog.SetLogLoggerLevel(slog.LevelError)
		} else if strings.HasPrefix("fatal", lv) {
			slog.SetLogLoggerLevel(LevelFatal)
		} else {
			return errors.New("The loglevel parameter value must be a prefix of one of theses words, \"trace\", \"debug\", \"info\", \"warning\", \"error\" or \"fatal\".")
		}
		return nil
	})
	pflag.StringVarP(&SchedulerFilename, "json", "j", "", "TBD")
	pflag.Lookup("json").NoOptDefVal = "scheduler.json"
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

	keylen += 3 // for output spacing
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

			fmt.Printf("%-*s DST: %-3s %+v Extend %s\n", keylen, name, SupportsDST(len(zone.Offsets)), zone.Aliases, zone.Extend)
			if len(description) != 0 {
				fmt.Println(description)
			}
		} else {
			fmt.Printf("Missing zone %s\n", name)
		}
		numAliases += len(zone.Aliases)
	}
	slog.Info("Statistics", "zoneinfos", len(zones), "aliases", numAliases, "total", len(zones)+numAliases)
}

// UsesDST checks if a given location observes Daylight Saving Time by comparing offsets.

func UsesDST(timezone string) (bool, string, string, int, error) {
	year := time.Now().Year()
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return false, "", "", year, err
	}

	// Check offset on a winter date (Jan 1) and a summer date (Jul 1)
	winterTime := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
	summerTime := time.Date(year, time.July, 1, 0, 0, 0, 0, loc)

	xst, winterOffset := winterTime.Zone()
	xdt, summerOffset := summerTime.Zone()

	// If the offsets are different, the timezone uses DST rules.
	return winterOffset != summerOffset, xst, xdt, year, nil
}

func GetOsTimeZones() ([]string, int) {
	var zoneDirs = []string{
		// Update path according to your OS
		"/usr/share/zoneinfo/",
		"/usr/share/lib/zoneinfo/",
		"/usr/lib/locale/TZ/",
	}

	for _, zd := range zoneDirs {
		walkTzDir(zd)
	}

	zones := make([]string, 0, len(TzInfos))
	keylen := 0
	for key, _ := range TzInfos {
		zones = append(zones, key)
		if len(key) > keylen {
			keylen = len(key)
		}
	}
	sort.Strings(zones)

	return zones, keylen
}

func walkTzDir(path string) {
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

		newPath := path + "/" + info.Name()

		if info.IsDir() {
			walkTzDir(newPath)
		} else {
			parts := strings.Split(newPath, "//")
			if len(parts) != 2 {
				continue
			}
			if zoneInfo, err := rfc9636.LoadLocation(parts[1], []string{parts[0]}); err == nil {
				slog.Debug("dump of zoneinfo", "timezone", parts[1])
				if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
					rfc9636.DumpLocation(zoneInfo)
				}

				if info.Type()&os.ModeSymlink != 0 {
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
					atz, found := strings.CutPrefix(resolvedPath, parts[0]+"/")
					if !found {
						slog.Error("Could not extract timezone alias", "path", resolvedPath)
						continue
					}
					slog.Debug("Timezone has alias", "timezone", atz, "alias", parts[1])
					TzInfos.AddZoneAlias(atz, parts[1])
				} else {
					TzInfos.Add(parts[1], zoneInfo)
				}

			} else {
				Trace("File is not a timezone file", "file", newPath)
			}

		}
	}
	return
}
