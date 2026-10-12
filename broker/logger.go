package broker

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// logAdapter sends the osbapi handler's structured logs to the scheduler's
// logger, with the fields written as sorted key=value pairs.
type logAdapter struct {
	log core.LogService
}

const logTag = "service-broker"

func (adapter logAdapter) Debug(msg string, fields map[string]any) {
	adapter.log.Debug(logTag, render(msg, fields))
}

func (adapter logAdapter) Info(msg string, fields map[string]any) {
	adapter.log.Info(logTag, render(msg, fields))
}

func (adapter logAdapter) Warn(msg string, fields map[string]any) {
	adapter.log.Warn(logTag, render(msg, fields))
}

func (adapter logAdapter) Error(msg string, fields map[string]any) {
	adapter.log.Error(logTag, render(msg, fields))
}

func render(msg string, fields map[string]any) string {
	if len(fields) == 0 {
		return msg
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	builder.WriteString(msg)
	for _, key := range keys {
		fmt.Fprintf(&builder, " %s=%v", key, fields[key])
	}

	return builder.String()
}
