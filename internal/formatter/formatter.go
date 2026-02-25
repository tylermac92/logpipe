// Package formatter provides output formatters that write log entries to an
// io.Writer in a human- or machine-readable form.
package formatter

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/tylermac92/logpipe/internal/parser"
)

// Formatter is the interface implemented by all output formatters.
// Format writes a single log entry to w and returns any write error.
type Formatter interface {
	Format(w io.Writer, entry parser.LogEntry) error
}

// JSONFormatter writes each log entry as a JSON object followed by a newline.
type JSONFormatter struct {
	// Pretty enables indented JSON output when true.
	Pretty bool
}

// Format marshals the entry to JSON and writes it to w. When Pretty is true
// the output is indented with two spaces; otherwise it is compact.
func (f *JSONFormatter) Format(w io.Writer, entry parser.LogEntry) error {
	var data []byte
	var err error

	if f.Pretty {
		data, err = json.MarshalIndent(entry, "", "  ")
	} else {
		data, err = json.Marshal(entry)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	_, err = w.Write(append(data, '\n'))
	return err
}

// ANSI escape codes used by TextFormatter for terminal coloring.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"  //nolint:unused
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// TextFormatter writes each log entry as a human-readable line of text in
// the format:
//
//	<timestamp> [LEVEL] <message> key=value ...
//
// Well-known field names (time/ts/timestamp, level/lvl/severity,
// message/msg/text) are pulled out and rendered in fixed positions; all
// remaining fields are appended as key=value pairs sorted alphabetically.
type TextFormatter struct {
	// Color enables ANSI terminal colours when true.
	Color bool
	// Fields restricts the extra key=value pairs to the named fields.
	// When empty, all non-canonical fields are printed.
	Fields []string
}

// Format writes a formatted text representation of entry to w.
func (f *TextFormatter) Format(w io.Writer, entry parser.LogEntry) error {
	timestamp := extractString(entry, "time", "ts", "timestamp")
	level := extractString(entry, "level", "lvl", "severity")
	message := extractString(entry, "message", "msg", "text")

	levelStr := f.colorizeLevel(level)
	timeStr := formatTimestamp(timestamp)

	// canonical holds the well-known field names that are rendered in fixed
	// positions so they are not duplicated in the trailing key=value pairs.
	canonical := map[string]bool{"time": true, "ts": true, "timestamp": true, "level": true, "lvl": true, "severity": true, "message": true, "msg": true, "text": true}

	var extras []string
	if len(f.Fields) > 0 {
		// User requested specific fields — render only those.
		for _, field := range f.Fields {
			if val, exists := entry[field]; exists {
				extras = append(extras, fmt.Sprintf("%s=%v", field, val))
			}
		}
	} else {
		// Render all non-canonical fields in sorted order for stable output.
		var keys []string
		for k := range entry {
			if !canonical[k] {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			extras = append(extras, fmt.Sprintf("%s=%v", k, entry[k]))
		}
	}

	extaStr := ""
	if len(extras) > 0 {
		if f.Color {
			extaStr = fmt.Sprintf(" %s%s%s", colorGray, strings.Join(extras, " "), colorReset)
		} else {
			extaStr = " " + strings.Join(extras, " ")
		}
	}

	_, err := fmt.Fprintf(w, "%s %s %s%s\n", timeStr, levelStr, message, extaStr)
	return err
}

// colorizeLevel returns the level string wrapped in ANSI colour codes when
// Color is enabled, or as a plain bracketed uppercase token otherwise.
func (f *TextFormatter) colorizeLevel(level string) string {
	if !f.Color {
		return fmt.Sprintf("[%-5s]", strings.ToUpper(level))
	}
	switch strings.ToLower(level) {
	case "error", "err", "fatal", "crit":
		return colorRed + colorBold + "[ERROR]" + colorReset
	case "warn", "warning":
		return colorYellow + colorBold + "[WARN ]" + colorReset
	case "info", "information":
		return colorGreen + colorBold + "[INFO ]" + colorReset
	default:
		return colorGray + "[" + strings.ToUpper(level) + "]" + colorReset
	}
}

// extractString tries each key in order and returns the string representation
// of the first one found in entry. Returns an empty string if none exist.
func extractString(entry parser.LogEntry, keys ...string) string {
	for _, key := range keys {
		if val, exists := entry[key]; exists {
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

// formatTimestamp normalises a raw timestamp string for display.
// It accepts:
//   - A Unix epoch (seconds, possibly fractional) greater than 1e9
//   - An RFC 3339 string
//   - Any other string, truncated to 15 characters
//
// Returns a fixed-width blank placeholder when value is empty.
func formatTimestamp(value string) string {
	if value == "" {
		return colorGray + "               " + colorReset
	}

	// Try to parse as a Unix timestamp (float).
	var f float64
	if _, err := fmt.Sscanf(value, "%f", &f); err == nil && f > 1e9 {
		t := time.Unix(int64(f), 0).UTC()
		return t.Format("15:04:05")
	}

	// Try RFC 3339 (e.g. "2024-01-15T12:34:56Z").
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.Format("15:04:05")
	}

	// Fall back to a prefix of the raw value.
	if len(value) > 15 {
		return value[:15]
	}
	return value
}

// LogfmtFormatter writes each log entry as a logfmt line: a sequence of
// space-separated key=value pairs sorted alphabetically by key. Values that
// contain spaces, tabs, or double-quotes are double-quoted with internal
// quotes escaped.
type LogfmtFormatter struct{}

// Format writes a logfmt representation of entry to w.
func (f *LogfmtFormatter) Format(w io.Writer, entry parser.LogEntry) error {
	var keys []string
	for k := range entry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		v := fmt.Sprintf("%v", entry[k])
		if strings.ContainsAny(v, " \t\"") {
			v = `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}

	_, err := fmt.Fprintln(w, strings.Join(parts, " "))
	return err
}
