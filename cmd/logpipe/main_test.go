package main

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/tylermac92/logpipe/internal/parser"
)

// =============================================================================
// multiFlag
// =============================================================================

func TestMultiFlag_String_Empty(t *testing.T) {
	m := multiFlag{}
	if got := m.String(); got != "" {
		t.Errorf("String() = %q, want empty string", got)
	}
}

func TestMultiFlag_String_Single(t *testing.T) {
	m := multiFlag{"level=error"}
	if got := m.String(); got != "level=error" {
		t.Errorf("String() = %q, want %q", got, "level=error")
	}
}

func TestMultiFlag_String_MultipleJoinedByComma(t *testing.T) {
	m := multiFlag{"level=error", "service=api", "region=us-east"}
	want := "level=error,service=api,region=us-east"
	if got := m.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestMultiFlag_Set_AppendsValue(t *testing.T) {
	var m multiFlag
	if err := m.Set("level=error"); err != nil {
		t.Fatalf("Set() returned unexpected error: %v", err)
	}
	if len(m) != 1 {
		t.Fatalf("expected len=1, got %d", len(m))
	}
	if m[0] != "level=error" {
		t.Errorf("m[0] = %q, want %q", m[0], "level=error")
	}
}

func TestMultiFlag_Set_AppendMultipleValues(t *testing.T) {
	var m multiFlag
	values := []string{"level=error", "service=api", "region=us-east"}
	for _, v := range values {
		if err := m.Set(v); err != nil {
			t.Fatalf("Set(%q) returned unexpected error: %v", v, err)
		}
	}
	if len(m) != 3 {
		t.Fatalf("expected len=3, got %d", len(m))
	}
	for i, want := range values {
		if m[i] != want {
			t.Errorf("m[%d] = %q, want %q", i, m[i], want)
		}
	}
}

func TestMultiFlag_Set_PreservesOrder(t *testing.T) {
	var m multiFlag
	m.Set("first")
	m.Set("second")
	m.Set("third")
	if m[0] != "first" || m[1] != "second" || m[2] != "third" {
		t.Errorf("values not in insertion order: %v", []string(m))
	}
}

func TestMultiFlag_Set_NeverReturnsError(t *testing.T) {
	var m multiFlag
	// The Set implementation always returns nil; any value is accepted.
	for _, v := range []string{"", "=", "invalid!!!", "level=error"} {
		if err := m.Set(v); err != nil {
			t.Errorf("Set(%q) returned unexpected error: %v", v, err)
		}
	}
}

func TestMultiFlag_String_RoundTrip(t *testing.T) {
	// Verify that String() reflects all values added via Set().
	var m multiFlag
	m.Set("a=1")
	m.Set("b=2")
	want := "a=1,b=2"
	if got := m.String(); got != want {
		t.Errorf("String() after two Set calls = %q, want %q", got, want)
	}
}

// multiFlag implements flag.Value; confirm it satisfies the interface at compile time.
func TestMultiFlag_ImplementsFlagValue(t *testing.T) {
	var m multiFlag
	// flag.Value requires String() string and Set(string) error.
	// Both are tested above; this test exists to document the contract.
	_ = m.String()
	_ = m.Set("x")
}

// =============================================================================
// sniffFormat
// =============================================================================

func TestSniffFormat_JSON(t *testing.T) {
	r := strings.NewReader(`{"level":"info","msg":"hello"}` + "\n")
	got, _, err := sniffFormat(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "json" {
		t.Errorf("got %q, want %q", got, "json")
	}
}

func TestSniffFormat_Logfmt(t *testing.T) {
	r := strings.NewReader("level=info msg=hello\n")
	got, _, err := sniffFormat(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "logfmt" {
		t.Errorf("got %q, want %q", got, "logfmt")
	}
}

func TestSniffFormat_LeadingBlankLines_JSON(t *testing.T) {
	r := strings.NewReader("\n\n\n" + `{"level":"warn"}` + "\n")
	got, _, err := sniffFormat(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "json" {
		t.Errorf("got %q, want %q", got, "json")
	}
}

func TestSniffFormat_LeadingBlankLines_Logfmt(t *testing.T) {
	r := strings.NewReader("\n\nlevel=error\n")
	got, _, err := sniffFormat(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "logfmt" {
		t.Errorf("got %q, want %q", got, "logfmt")
	}
}

func TestSniffFormat_EmptyInput_DefaultsToJSON(t *testing.T) {
	r := strings.NewReader("")
	got, _, err := sniffFormat(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "json" {
		t.Errorf("got %q, want %q", got, "json")
	}
}

func TestSniffFormat_WhitespaceOnly_DefaultsToJSON(t *testing.T) {
	r := strings.NewReader("   \n  \n")
	got, _, err := sniffFormat(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "json" {
		t.Errorf("got %q, want %q", got, "json")
	}
}

func TestSniffFormat_ReconstructedReaderContainsSniffedLine(t *testing.T) {
	input := `{"level":"info","msg":"hello"}` + "\n"
	r := strings.NewReader(input)
	_, reconstructed, err := sniffFormat(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := io.ReadAll(reconstructed)
	if err != nil {
		t.Fatalf("reading reconstructed reader: %v", err)
	}
	if string(got) != input {
		t.Errorf("reconstructed reader = %q, want %q", string(got), input)
	}
}

func TestSniffFormat_ReconstructedReaderContainsAllLines(t *testing.T) {
	input := "level=info msg=first\nlevel=error msg=second\n"
	r := strings.NewReader(input)
	_, reconstructed, err := sniffFormat(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := io.ReadAll(reconstructed)
	if err != nil {
		t.Fatalf("reading reconstructed reader: %v", err)
	}
	if string(got) != input {
		t.Errorf("reconstructed reader = %q, want %q", string(got), input)
	}
}

// =============================================================================
// collectStats
// =============================================================================

// matchAll is a match function that accepts every entry.
func matchAll(_ parser.LogEntry) bool { return true }

// makeEntries returns a closed channel pre-loaded with the given entries.
func makeEntries(entries ...parser.LogEntry) <-chan parser.LogEntry {
	ch := make(chan parser.LogEntry, len(entries))
	for _, e := range entries {
		ch <- e
	}
	close(ch)
	return ch
}

func TestCollectStats_CountsByValue(t *testing.T) {
	ch := makeEntries(
		parser.LogEntry{"level": "info"},
		parser.LogEntry{"level": "error"},
		parser.LogEntry{"level": "info"},
	)
	got := collectStats(ch, matchAll, "level")
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(got), got)
	}
	if got[0].Value != "info" || got[0].Count != 2 {
		t.Errorf("got[0] = %+v, want {info 2}", got[0])
	}
	if got[1].Value != "error" || got[1].Count != 1 {
		t.Errorf("got[1] = %+v, want {error 1}", got[1])
	}
}

func TestCollectStats_SortedByCountDescending(t *testing.T) {
	ch := makeEntries(
		parser.LogEntry{"level": "error"},
		parser.LogEntry{"level": "info"},
		parser.LogEntry{"level": "info"},
		parser.LogEntry{"level": "info"},
		parser.LogEntry{"level": "warn"},
		parser.LogEntry{"level": "warn"},
	)
	got := collectStats(ch, matchAll, "level")
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[0].Value != "info" || got[0].Count != 3 {
		t.Errorf("got[0] = %+v, want {info 3}", got[0])
	}
	if got[1].Value != "warn" || got[1].Count != 2 {
		t.Errorf("got[1] = %+v, want {warn 2}", got[1])
	}
	if got[2].Value != "error" || got[2].Count != 1 {
		t.Errorf("got[2] = %+v, want {error 1}", got[2])
	}
}

func TestCollectStats_TiesBrokenAlphabetically(t *testing.T) {
	ch := makeEntries(
		parser.LogEntry{"svc": "zebra"},
		parser.LogEntry{"svc": "alpha"},
		parser.LogEntry{"svc": "middle"},
	)
	got := collectStats(ch, matchAll, "svc")
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	// All counts are 1, so alphabetical order applies.
	if got[0].Value != "alpha" {
		t.Errorf("got[0].Value = %q, want %q", got[0].Value, "alpha")
	}
	if got[1].Value != "middle" {
		t.Errorf("got[1].Value = %q, want %q", got[1].Value, "middle")
	}
	if got[2].Value != "zebra" {
		t.Errorf("got[2].Value = %q, want %q", got[2].Value, "zebra")
	}
}

func TestCollectStats_MissingFieldCountedAsNone(t *testing.T) {
	ch := makeEntries(
		parser.LogEntry{"level": "info"},
		parser.LogEntry{"msg": "no level field"},
	)
	got := collectStats(ch, matchAll, "level")
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(got), got)
	}
	found := false
	for _, s := range got {
		if s.Value == "(none)" && s.Count == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected (none): 1 in results, got %v", got)
	}
}

func TestCollectStats_FilterApplied(t *testing.T) {
	ch := makeEntries(
		parser.LogEntry{"level": "info", "svc": "api"},
		parser.LogEntry{"level": "error", "svc": "db"},
		parser.LogEntry{"level": "error", "svc": "api"},
	)
	onlyErrors := func(e parser.LogEntry) bool {
		return e["level"] == "error"
	}
	got := collectStats(ch, onlyErrors, "svc")
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(got), got)
	}
	for _, s := range got {
		if s.Count != 1 {
			t.Errorf("expected count 1 for %q, got %d", s.Value, s.Count)
		}
	}
}

func TestCollectStats_EmptyInput(t *testing.T) {
	ch := makeEntries()
	got := collectStats(ch, matchAll, "level")
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

// =============================================================================
// parseTimestampForSort
// =============================================================================

func TestParseTimestampForSort_RFC3339(t *testing.T) {
	entry := parser.LogEntry{"time": "2024-01-15T12:34:56Z"}
	got := parseTimestampForSort(entry)
	want, _ := time.Parse(time.RFC3339, "2024-01-15T12:34:56Z")
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseTimestampForSort_UnixEpoch(t *testing.T) {
	entry := parser.LogEntry{"time": "1704067200"}
	got := parseTimestampForSort(entry)
	want := time.Unix(1704067200, 0).UTC()
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseTimestampForSort_AlternativeKey_Ts(t *testing.T) {
	entry := parser.LogEntry{"ts": "2024-06-01T00:00:00Z"}
	got := parseTimestampForSort(entry)
	if got.IsZero() {
		t.Error("expected non-zero time for ts key")
	}
}

func TestParseTimestampForSort_AlternativeKey_Timestamp(t *testing.T) {
	entry := parser.LogEntry{"timestamp": "2024-06-01T00:00:00Z"}
	got := parseTimestampForSort(entry)
	if got.IsZero() {
		t.Error("expected non-zero time for timestamp key")
	}
}

func TestParseTimestampForSort_NoTimestampField_ReturnsZero(t *testing.T) {
	entry := parser.LogEntry{"level": "info", "msg": "hello"}
	got := parseTimestampForSort(entry)
	if !got.IsZero() {
		t.Errorf("expected zero time, got %v", got)
	}
}

func TestParseTimestampForSort_UnparsableValue_ReturnsZero(t *testing.T) {
	entry := parser.LogEntry{"time": "not-a-timestamp"}
	got := parseTimestampForSort(entry)
	if !got.IsZero() {
		t.Errorf("expected zero time for unparseable value, got %v", got)
	}
}

// =============================================================================
// loadEntries
// =============================================================================

func TestLoadEntries_TagsSource(t *testing.T) {
	r := strings.NewReader(`{"level":"info"}` + "\n")
	got := loadEntries(r, parser.NewJSONParser(), "myfile.log")
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0].entry["_source"] != "myfile.log" {
		t.Errorf("_source = %q, want %q", got[0].entry["_source"], "myfile.log")
	}
}

func TestLoadEntries_ParsesTimestamp(t *testing.T) {
	r := strings.NewReader(`{"time":"2024-03-01T10:00:00Z","level":"info"}` + "\n")
	got := loadEntries(r, parser.NewJSONParser(), "svc.log")
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	want, _ := time.Parse(time.RFC3339, "2024-03-01T10:00:00Z")
	if !got[0].t.Equal(want) {
		t.Errorf("t = %v, want %v", got[0].t, want)
	}
}

func TestLoadEntries_MultipleEntries(t *testing.T) {
	r := strings.NewReader(`{"level":"info"}` + "\n" + `{"level":"error"}` + "\n")
	got := loadEntries(r, parser.NewJSONParser(), "app.log")
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
}

func TestLoadEntries_EmptyReader(t *testing.T) {
	r := strings.NewReader("")
	got := loadEntries(r, parser.NewJSONParser(), "empty.log")
	if len(got) != 0 {
		t.Errorf("expected 0 entries, got %d", len(got))
	}
}
