package formatter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tylermac92/logpipe/internal/parser"
)

// =============================================================================
// JSONFormatter
// =============================================================================

func TestJSONFormatter_NonPretty_ValidJSON(t *testing.T) {
	f := &JSONFormatter{Pretty: false}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"level": "info", "msg": "hello"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	line := strings.TrimSpace(buf.String())
	var result map[string]any
	if err := json.Unmarshal([]byte(line), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, line)
	}
}

func TestJSONFormatter_NonPretty_SingleLine(t *testing.T) {
	f := &JSONFormatter{Pretty: false}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"level": "info", "msg": "hello"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Trailing newline is appended, but the JSON itself must be on one line.
	trimmed := strings.TrimRight(buf.String(), "\n")
	if strings.ContainsRune(trimmed, '\n') {
		t.Errorf("non-pretty JSON must be a single line, got: %s", buf.String())
	}
}

func TestJSONFormatter_Pretty_ValidJSON(t *testing.T) {
	f := &JSONFormatter{Pretty: true}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"level": "info", "msg": "hello"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &result); err != nil {
		t.Fatalf("pretty output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
}

func TestJSONFormatter_Pretty_ContainsNewlines(t *testing.T) {
	f := &JSONFormatter{Pretty: true}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"a": "1", "b": "2"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "\n") {
		t.Error("pretty JSON should contain newlines")
	}
}

func TestJSONFormatter_Pretty_ContainsIndentation(t *testing.T) {
	f := &JSONFormatter{Pretty: true}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"key": "val"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "  ") {
		t.Error("pretty JSON should contain indentation (two spaces)")
	}
}

func TestJSONFormatter_TrailingNewline(t *testing.T) {
	for _, pretty := range []bool{false, true} {
		f := &JSONFormatter{Pretty: pretty}
		var buf bytes.Buffer
		if err := f.Format(&buf, parser.LogEntry{"k": "v"}); err != nil {
			t.Fatalf("Pretty=%v: unexpected error: %v", pretty, err)
		}
		if !strings.HasSuffix(buf.String(), "\n") {
			t.Errorf("Pretty=%v: output should end with newline, got: %q", pretty, buf.String())
		}
	}
}

func TestJSONFormatter_EmptyEntry(t *testing.T) {
	f := &JSONFormatter{Pretty: false}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "{}" {
		t.Errorf("expected {}, got: %s", strings.TrimSpace(buf.String()))
	}
}

func TestJSONFormatter_AllFieldsPreserved(t *testing.T) {
	f := &JSONFormatter{Pretty: false}
	entry := parser.LogEntry{"a": "1", "b": float64(2), "c": true}
	var buf bytes.Buffer
	if err := f.Format(&buf, entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 fields, got %d: %v", len(result), result)
	}
	if result["a"] != "1" {
		t.Errorf("a: got %v, want 1", result["a"])
	}
	if result["b"] != float64(2) {
		t.Errorf("b: got %v, want 2", result["b"])
	}
	if result["c"] != true {
		t.Errorf("c: got %v, want true", result["c"])
	}
}

// =============================================================================
// TextFormatter
// =============================================================================

func TestTextFormatter_BasicOutput_ContainsMessage(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	err := f.Format(&buf, parser.LogEntry{"level": "info", "msg": "test message", "time": "2024-01-01T12:00:00Z"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "test message") {
		t.Errorf("output should contain message, got: %s", buf.String())
	}
}

func TestTextFormatter_BasicOutput_ContainsLevel(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	err := f.Format(&buf, parser.LogEntry{"level": "info", "msg": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "INFO") {
		t.Errorf("output should contain level, got: %s", buf.String())
	}
}

func TestTextFormatter_TrailingNewline(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"level": "info", "msg": "test"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Errorf("output should end with newline, got: %q", buf.String())
	}
}

func TestTextFormatter_ColorDisabled_LevelFormat_Info(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "info", "msg": "x"})
	if !strings.Contains(buf.String(), "[INFO ]") {
		t.Errorf("expected [INFO ] in output, got: %s", buf.String())
	}
}

func TestTextFormatter_ColorDisabled_LevelFormat_Error(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "error", "msg": "x"})
	if !strings.Contains(buf.String(), "[ERROR]") {
		t.Errorf("expected [ERROR] in output, got: %s", buf.String())
	}
}

func TestTextFormatter_ColorDisabled_LevelFormat_Warn(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "warn", "msg": "x"})
	if !strings.Contains(buf.String(), "[WARN ]") {
		t.Errorf("expected [WARN ] in output, got: %s", buf.String())
	}
}

func TestTextFormatter_ColorDisabled_LevelFormat_Debug(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "debug", "msg": "x"})
	if !strings.Contains(buf.String(), "[DEBUG]") {
		t.Errorf("expected [DEBUG] in output, got: %s", buf.String())
	}
}

func TestTextFormatter_ColorDisabled_LevelFormat_UpperCase(t *testing.T) {
	// Level is uppercased regardless of input case.
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "INFO", "msg": "x"})
	if !strings.Contains(buf.String(), "[INFO ]") {
		t.Errorf("expected [INFO ] for uppercase input, got: %s", buf.String())
	}
}

func TestTextFormatter_ColorEnabled_ContainsANSICodes(t *testing.T) {
	f := &TextFormatter{Color: true}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"level": "error", "msg": "test"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "\033[") {
		t.Error("expected ANSI escape codes in color output")
	}
}

func TestTextFormatter_ColorEnabled_ErrorLevel_UsesRed(t *testing.T) {
	f := &TextFormatter{Color: true}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "error", "msg": "x"})
	if !strings.Contains(buf.String(), colorRed) {
		t.Errorf("expected red color code for error level, got: %q", buf.String())
	}
}

func TestTextFormatter_ColorEnabled_WarnLevel_UsesYellow(t *testing.T) {
	f := &TextFormatter{Color: true}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "warn", "msg": "x"})
	if !strings.Contains(buf.String(), colorYellow) {
		t.Errorf("expected yellow color code for warn level, got: %q", buf.String())
	}
}

func TestTextFormatter_ColorEnabled_InfoLevel_UsesGreen(t *testing.T) {
	f := &TextFormatter{Color: true}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "info", "msg": "x"})
	if !strings.Contains(buf.String(), colorGreen) {
		t.Errorf("expected green color code for info level, got: %q", buf.String())
	}
}

func TestTextFormatter_ColorEnabled_UnknownLevel_UsesGray(t *testing.T) {
	f := &TextFormatter{Color: true}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "trace", "msg": "x"})
	if !strings.Contains(buf.String(), colorGray) {
		t.Errorf("expected gray color code for unknown level, got: %q", buf.String())
	}
}

// colorizeLevel aliases for error: "err", "fatal", "crit"
func TestTextFormatter_ColorEnabled_ErrAlias(t *testing.T) {
	f := &TextFormatter{Color: true}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "err", "msg": "x"})
	if !strings.Contains(buf.String(), "[ERROR]") {
		t.Errorf("expected [ERROR] for level=err, got: %q", buf.String())
	}
}

func TestTextFormatter_ColorEnabled_FatalAlias(t *testing.T) {
	f := &TextFormatter{Color: true}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "fatal", "msg": "x"})
	if !strings.Contains(buf.String(), "[ERROR]") {
		t.Errorf("expected [ERROR] for level=fatal, got: %q", buf.String())
	}
}

func TestTextFormatter_ColorEnabled_WarningAlias(t *testing.T) {
	f := &TextFormatter{Color: true}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "warning", "msg": "x"})
	if !strings.Contains(buf.String(), "[WARN ]") {
		t.Errorf("expected [WARN ] for level=warning, got: %q", buf.String())
	}
}

func TestTextFormatter_ColorEnabled_InformationAlias(t *testing.T) {
	f := &TextFormatter{Color: true}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "information", "msg": "x"})
	if !strings.Contains(buf.String(), "[INFO ]") {
		t.Errorf("expected [INFO ] for level=information, got: %q", buf.String())
	}
}

func TestTextFormatter_MissingLevel_NoError(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"msg": "no level here"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "no level here") {
		t.Errorf("output should contain message, got: %s", buf.String())
	}
}

func TestTextFormatter_MissingMessage_NoError(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"level": "info"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTextFormatter_MissingTimestamp_NoError(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"level": "info", "msg": "hello"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Alternative keys for level: "lvl", "severity"
func TestTextFormatter_AlternativeLevelKey_Lvl(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"lvl": "warn", "msg": "x"})
	if !strings.Contains(buf.String(), "WARN") {
		t.Errorf("expected WARN from lvl key, got: %s", buf.String())
	}
}

func TestTextFormatter_AlternativeLevelKey_Severity(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"severity": "error", "msg": "x"})
	if !strings.Contains(buf.String(), "ERROR") {
		t.Errorf("expected ERROR from severity key, got: %s", buf.String())
	}
}

// Alternative keys for message: "message", "text"
func TestTextFormatter_AlternativeMessageKey_Message(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "info", "message": "msg via message key"})
	if !strings.Contains(buf.String(), "msg via message key") {
		t.Errorf("expected message content, got: %s", buf.String())
	}
}

func TestTextFormatter_AlternativeMessageKey_Text(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "info", "text": "msg via text key"})
	if !strings.Contains(buf.String(), "msg via text key") {
		t.Errorf("expected message content, got: %s", buf.String())
	}
}

// Alternative keys for timestamp: "ts", "timestamp"
func TestTextFormatter_AlternativeTimeKey_Ts(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "info", "msg": "x", "ts": "2024-06-15T09:30:00Z"})
	if !strings.Contains(buf.String(), "09:30:00") {
		t.Errorf("expected formatted time from ts key, got: %s", buf.String())
	}
}

func TestTextFormatter_AlternativeTimeKey_Timestamp(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "info", "msg": "x", "timestamp": "2024-06-15T14:45:00Z"})
	if !strings.Contains(buf.String(), "14:45:00") {
		t.Errorf("expected formatted time from timestamp key, got: %s", buf.String())
	}
}

func TestTextFormatter_RFC3339Timestamp_FormattedAsTime(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "info", "msg": "x", "time": "2024-01-01T12:34:56Z"})
	if !strings.Contains(buf.String(), "12:34:56") {
		t.Errorf("expected 12:34:56 in output, got: %s", buf.String())
	}
}

func TestTextFormatter_UnixTimestamp_FormattedAsTime(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	// 1704067200 = 2024-01-01T00:00:00Z
	f.Format(&buf, parser.LogEntry{"level": "info", "msg": "x", "time": "1704067200"})
	out := buf.String()
	// Should contain a HH:MM:SS formatted time.
	if !strings.Contains(out, ":") {
		t.Errorf("expected HH:MM:SS formatted time, got: %s", out)
	}
}

// Non-canonical extra fields are appended after message.
func TestTextFormatter_ExtrasAppended_NoFields(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{
		"level":   "info",
		"msg":     "hello",
		"service": "api",
		"host":    "srv1",
	})
	out := buf.String()
	if !strings.Contains(out, "service=api") {
		t.Errorf("expected service=api in extras, got: %s", out)
	}
	if !strings.Contains(out, "host=srv1") {
		t.Errorf("expected host=srv1 in extras, got: %s", out)
	}
}

func TestTextFormatter_ExtrasSortedAlphabetically(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{
		"level":   "info",
		"msg":     "hello",
		"z_field": "last",
		"a_field": "first",
		"m_field": "middle",
	})
	out := buf.String()
	aIdx := strings.Index(out, "a_field")
	mIdx := strings.Index(out, "m_field")
	zIdx := strings.Index(out, "z_field")
	if aIdx == -1 || mIdx == -1 || zIdx == -1 {
		t.Fatalf("expected all fields in output, got: %s", out)
	}
	if !(aIdx < mIdx && mIdx < zIdx) {
		t.Errorf("fields should be sorted alphabetically (a < m < z), got: %s", out)
	}
}

func TestTextFormatter_CanonicalFieldsNotInExtras(t *testing.T) {
	f := &TextFormatter{Color: false}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{
		"time":  "2024-01-01T00:00:00Z",
		"level": "info",
		"msg":   "hello",
	})
	out := buf.String()
	// Canonical fields must not appear as "key=value" extras.
	for _, bad := range []string{"time=", "level=", "msg="} {
		if strings.Contains(out, bad) {
			t.Errorf("canonical field %q must not appear in extras, got: %s", bad, out)
		}
	}
}

// When Fields is specified, only those fields appear as extras (in order).
func TestTextFormatter_FieldsFilter_OnlyIncludesSpecified(t *testing.T) {
	f := &TextFormatter{Color: false, Fields: []string{"service"}}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{
		"level":   "info",
		"msg":     "hello",
		"service": "api",
		"host":    "srv1",
	})
	out := buf.String()
	if !strings.Contains(out, "service=api") {
		t.Errorf("expected service=api in output, got: %s", out)
	}
	if strings.Contains(out, "host=") {
		t.Errorf("host should not appear when not in Fields, got: %s", out)
	}
}

func TestTextFormatter_FieldsFilter_AbsentFieldsSkipped(t *testing.T) {
	// Requesting a field that doesn't exist in the entry should not error.
	f := &TextFormatter{Color: false, Fields: []string{"nonexistent"}}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"level": "info", "msg": "hello"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "nonexistent" is absent, so no extras appear.
	if strings.Contains(buf.String(), "nonexistent") {
		t.Errorf("absent field should not appear in output, got: %s", buf.String())
	}
}

func TestTextFormatter_FieldsFilter_MultipleFields(t *testing.T) {
	f := &TextFormatter{Color: false, Fields: []string{"service", "region"}}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{
		"level":   "info",
		"msg":     "hello",
		"service": "api",
		"region":  "us-east",
		"host":    "srv1",
	})
	out := buf.String()
	if !strings.Contains(out, "service=api") {
		t.Errorf("expected service=api, got: %s", out)
	}
	if !strings.Contains(out, "region=us-east") {
		t.Errorf("expected region=us-east, got: %s", out)
	}
	if strings.Contains(out, "host=") {
		t.Errorf("host should be excluded, got: %s", out)
	}
}

func TestTextFormatter_ColorEnabled_ExtrasInGray(t *testing.T) {
	f := &TextFormatter{Color: true}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "info", "msg": "x", "svc": "api"})
	out := buf.String()
	// The extras section is wrapped in gray.
	if !strings.Contains(out, colorGray) {
		t.Errorf("expected gray ANSI code for extras in color mode, got: %q", out)
	}
}

// =============================================================================
// LogfmtFormatter
// =============================================================================

func TestLogfmtFormatter_BasicOutput_ContainsKeyValues(t *testing.T) {
	f := &LogfmtFormatter{}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"level": "info", "msg": "hello"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if !strings.Contains(out, "level=info") {
		t.Errorf("expected level=info, got: %s", out)
	}
	if !strings.Contains(out, "msg=hello") {
		t.Errorf("expected msg=hello, got: %s", out)
	}
}

func TestLogfmtFormatter_TrailingNewline(t *testing.T) {
	f := &LogfmtFormatter{}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{"k": "v"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Errorf("output should end with newline, got: %q", buf.String())
	}
}

func TestLogfmtFormatter_KeysSortedAlphabetically(t *testing.T) {
	f := &LogfmtFormatter{}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"z_key": "last", "a_key": "first", "m_key": "middle"})
	out := buf.String()
	aIdx := strings.Index(out, "a_key")
	mIdx := strings.Index(out, "m_key")
	zIdx := strings.Index(out, "z_key")
	if aIdx == -1 || mIdx == -1 || zIdx == -1 {
		t.Fatalf("expected all keys in output, got: %s", out)
	}
	if !(aIdx < mIdx && mIdx < zIdx) {
		t.Errorf("keys should be sorted alphabetically, got: %s", out)
	}
}

func TestLogfmtFormatter_PlainValue_NotQuoted(t *testing.T) {
	f := &LogfmtFormatter{}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "error"})
	out := strings.TrimSpace(buf.String())
	if out != "level=error" {
		t.Errorf("expected level=error, got: %s", out)
	}
}

func TestLogfmtFormatter_ValueWithSpace_IsQuoted(t *testing.T) {
	f := &LogfmtFormatter{}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"msg": "hello world"})
	out := buf.String()
	if !strings.Contains(out, `msg="hello world"`) {
		t.Errorf("expected quoted value for space, got: %s", out)
	}
}

func TestLogfmtFormatter_ValueWithTab_IsQuoted(t *testing.T) {
	f := &LogfmtFormatter{}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"msg": "hello\tworld"})
	out := buf.String()
	if !strings.Contains(out, `"`) {
		t.Errorf("expected quoted value for tab, got: %s", out)
	}
}

func TestLogfmtFormatter_ValueWithDoubleQuote_IsQuotedAndEscaped(t *testing.T) {
	f := &LogfmtFormatter{}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"msg": `say "hello"`})
	out := buf.String()
	// The value contains quotes, so the whole value is wrapped in quotes
	// and inner quotes are backslash-escaped.
	if !strings.Contains(out, `msg="say \"hello\""`) {
		t.Errorf("expected escaped quotes, got: %s", out)
	}
}

func TestLogfmtFormatter_EmptyEntry_OutputsBlankLine(t *testing.T) {
	f := &LogfmtFormatter{}
	var buf bytes.Buffer
	if err := f.Format(&buf, parser.LogEntry{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty entry → empty parts → Fprintln writes just a newline.
	if buf.String() != "\n" {
		t.Errorf("expected single newline for empty entry, got: %q", buf.String())
	}
}

func TestLogfmtFormatter_MultipleEntries_EachOnOwnLine(t *testing.T) {
	f := &LogfmtFormatter{}
	var buf bytes.Buffer
	f.Format(&buf, parser.LogEntry{"level": "info", "msg": "first"})
	f.Format(&buf, parser.LogEntry{"level": "error", "msg": "second"})
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %v", len(lines), lines)
	}
}

// =============================================================================
// formatTimestamp (white-box tests: package formatter)
// =============================================================================

func TestFormatTimestamp_EmptyString_ReturnsPlaceholder(t *testing.T) {
	out := formatTimestamp("")
	// Returns colorGray + 15 spaces + colorReset — non-empty.
	if out == "" {
		t.Error("expected non-empty placeholder for empty timestamp")
	}
	if !strings.Contains(out, colorGray) {
		t.Errorf("expected gray color code in placeholder, got: %q", out)
	}
}

func TestFormatTimestamp_RFC3339_FormattedAsHHMMSS(t *testing.T) {
	out := formatTimestamp("2024-01-15T09:30:00Z")
	if out != "09:30:00" {
		t.Errorf("got %q, want %q", out, "09:30:00")
	}
}

func TestFormatTimestamp_RFC3339_WithOffset(t *testing.T) {
	// time.Parse(time.RFC3339, ...) normalizes to the parsed zone; Format("15:04:05")
	// outputs in that zone. UTC offset "+00:00" should give same as "Z".
	out := formatTimestamp("2024-06-01T18:00:00+00:00")
	if out != "18:00:00" {
		t.Errorf("got %q, want %q", out, "18:00:00")
	}
}

func TestFormatTimestamp_UnixSeconds_FormattedAsHHMMSS(t *testing.T) {
	// 1704067200 = 2024-01-01T00:00:00Z
	out := formatTimestamp("1704067200")
	if out != "00:00:00" {
		t.Errorf("got %q, want %q", out, "00:00:00")
	}
}

func TestFormatTimestamp_UnixFloat_FormattedAsHHMMSS(t *testing.T) {
	// Float unix timestamp; fractional seconds are truncated.
	out := formatTimestamp("1704067200.5")
	if out != "00:00:00" {
		t.Errorf("got %q, want %q", out, "00:00:00")
	}
}

func TestFormatTimestamp_SmallNumber_NotTreatedAsUnix(t *testing.T) {
	// Numbers <= 1e9 are not treated as unix timestamps.
	// "123" is a short string (len <= 15) and cannot be parsed as RFC3339,
	// and 123.0 <= 1e9, so it falls through to the string truncation path.
	out := formatTimestamp("123")
	if out != "123" {
		t.Errorf("got %q, want %q", out, "123")
	}
}

func TestFormatTimestamp_ShortNonParseable_ReturnedAsIs(t *testing.T) {
	out := formatTimestamp("short")
	if out != "short" {
		t.Errorf("got %q, want %q", out, "short")
	}
}

func TestFormatTimestamp_ExactlyFifteenChars_ReturnedAsIs(t *testing.T) {
	// Use a non-numeric string that can't be parsed as a float or RFC3339,
	// so it reaches the len-check branch. Exactly 15 chars → returned as-is.
	val := "abcdefghijklmno" // exactly 15 chars, not a number, not RFC3339
	out := formatTimestamp(val)
	if out != val {
		t.Errorf("got %q, want %q", out, val)
	}
}

func TestFormatTimestamp_MoreThanFifteenChars_Truncated(t *testing.T) {
	val := "this-is-a-very-long-non-parseable-timestamp"
	out := formatTimestamp(val)
	if len(out) > 15 {
		t.Errorf("expected truncation to 15 chars, got %d: %q", len(out), out)
	}
	if out != val[:15] {
		t.Errorf("got %q, want %q", out, val[:15])
	}
}

// =============================================================================
// extractString (white-box tests: package formatter)
// =============================================================================

func TestExtractString_FirstKeyPresent(t *testing.T) {
	entry := parser.LogEntry{"time": "2024", "ts": "old"}
	out := extractString(entry, "time", "ts")
	if out != "2024" {
		t.Errorf("got %q, want %q", out, "2024")
	}
}

func TestExtractString_FallsBackToSecondKey(t *testing.T) {
	entry := parser.LogEntry{"ts": "fallback"}
	out := extractString(entry, "time", "ts")
	if out != "fallback" {
		t.Errorf("got %q, want %q", out, "fallback")
	}
}

func TestExtractString_NoKeyPresent_ReturnsEmpty(t *testing.T) {
	entry := parser.LogEntry{"other": "value"}
	out := extractString(entry, "time", "ts", "timestamp")
	if out != "" {
		t.Errorf("got %q, want empty string", out)
	}
}

func TestExtractString_EmptyEntry_ReturnsEmpty(t *testing.T) {
	out := extractString(parser.LogEntry{}, "level", "lvl")
	if out != "" {
		t.Errorf("got %q, want empty string", out)
	}
}

func TestExtractString_NumericValue_ReturnedAsString(t *testing.T) {
	entry := parser.LogEntry{"count": float64(42)}
	out := extractString(entry, "count")
	if out != "42" {
		t.Errorf("got %q, want %q", out, "42")
	}
}

func TestExtractString_BooleanValue_ReturnedAsString(t *testing.T) {
	entry := parser.LogEntry{"ok": true}
	out := extractString(entry, "ok")
	if out != "true" {
		t.Errorf("got %q, want %q", out, "true")
	}
}

// =============================================================================
// colorizeLevel (white-box tests: unexported method on TextFormatter)
// =============================================================================

func TestColorizeLevel_NoColor_PadsToFiveChars(t *testing.T) {
	f := &TextFormatter{Color: false}
	tests := []struct {
		level    string
		expected string
	}{
		{"info", "[INFO ]"},
		{"error", "[ERROR]"},
		{"warn", "[WARN ]"},
		{"debug", "[DEBUG]"},
		{"", "[     ]"},
		{"x", "[X    ]"},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := f.colorizeLevel(tt.level)
			if got != tt.expected {
				t.Errorf("colorizeLevel(%q) = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

func TestColorizeLevel_Color_ErrorGroup(t *testing.T) {
	f := &TextFormatter{Color: true}
	for _, level := range []string{"error", "err", "fatal", "crit"} {
		got := f.colorizeLevel(level)
		if !strings.Contains(got, "[ERROR]") {
			t.Errorf("colorizeLevel(%q) should produce [ERROR], got: %q", level, got)
		}
		if !strings.Contains(got, colorRed) {
			t.Errorf("colorizeLevel(%q) should use red, got: %q", level, got)
		}
	}
}

func TestColorizeLevel_Color_WarnGroup(t *testing.T) {
	f := &TextFormatter{Color: true}
	for _, level := range []string{"warn", "warning"} {
		got := f.colorizeLevel(level)
		if !strings.Contains(got, "[WARN ]") {
			t.Errorf("colorizeLevel(%q) should produce [WARN ], got: %q", level, got)
		}
		if !strings.Contains(got, colorYellow) {
			t.Errorf("colorizeLevel(%q) should use yellow, got: %q", level, got)
		}
	}
}

func TestColorizeLevel_Color_InfoGroup(t *testing.T) {
	f := &TextFormatter{Color: true}
	for _, level := range []string{"info", "information"} {
		got := f.colorizeLevel(level)
		if !strings.Contains(got, "[INFO ]") {
			t.Errorf("colorizeLevel(%q) should produce [INFO ], got: %q", level, got)
		}
		if !strings.Contains(got, colorGreen) {
			t.Errorf("colorizeLevel(%q) should use green, got: %q", level, got)
		}
	}
}

func TestColorizeLevel_Color_UnknownLevel_UsesGrayAndUpperCase(t *testing.T) {
	f := &TextFormatter{Color: true}
	got := f.colorizeLevel("trace")
	if !strings.Contains(got, "[TRACE]") {
		t.Errorf("expected [TRACE] for unknown level, got: %q", got)
	}
	if !strings.Contains(got, colorGray) {
		t.Errorf("expected gray for unknown level, got: %q", got)
	}
}

func TestColorizeLevel_Color_CaseInsensitive(t *testing.T) {
	f := &TextFormatter{Color: true}
	lower := f.colorizeLevel("error")
	upper := f.colorizeLevel("ERROR")
	if lower != upper {
		t.Errorf("colorizeLevel should be case-insensitive: %q != %q", lower, upper)
	}
}
