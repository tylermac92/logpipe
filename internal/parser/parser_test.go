package parser

import (
	"strings"
	"testing"
)

// collectEntries drains both channels concurrently and returns all entries and errors.
func collectEntries(t *testing.T, entries <-chan LogEntry, errors <-chan error) ([]LogEntry, []error) {
	t.Helper()
	var got []LogEntry
	var errs []error

	errsDone := make(chan struct{})
	go func() {
		defer close(errsDone)
		for err := range errors {
			errs = append(errs, err)
		}
	}()

	for entry := range entries {
		got = append(got, entry)
	}
	<-errsDone
	return got, errs
}

// r is a convenience helper to create a *strings.Reader.
func r(s string) *strings.Reader {
	return strings.NewReader(s)
}

// =============================================================================
// JSONParser
// =============================================================================

func TestJSONParser_SingleValidEntry(t *testing.T) {
	p := NewJSONParser()
	entries, errs := p.Parse(r(`{"level":"info","msg":"hello"}`))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0]["level"] != "info" {
		t.Errorf("level: got %v, want info", got[0]["level"])
	}
	if got[0]["msg"] != "hello" {
		t.Errorf("msg: got %v, want hello", got[0]["msg"])
	}
}

func TestJSONParser_MultipleEntries(t *testing.T) {
	input := `{"level":"info","msg":"first"}
{"level":"error","msg":"second"}
{"level":"warn","msg":"third"}`
	p := NewJSONParser()
	entries, errs := p.Parse(r(input))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[0]["msg"] != "first" {
		t.Errorf("entry 0 msg: got %v, want first", got[0]["msg"])
	}
	if got[2]["msg"] != "third" {
		t.Errorf("entry 2 msg: got %v, want third", got[2]["msg"])
	}
}

func TestJSONParser_EmptyReader(t *testing.T) {
	p := NewJSONParser()
	entries, errs := p.Parse(r(""))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(got))
	}
}

func TestJSONParser_SkipsEmptyLines(t *testing.T) {
	input := `{"level":"info","msg":"a"}

{"level":"error","msg":"b"}

`
	p := NewJSONParser()
	entries, errs := p.Parse(r(input))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
}

func TestJSONParser_SkipsWhitespaceOnlyLines(t *testing.T) {
	input := "   \n{\"level\":\"info\"}\n   \n"
	p := NewJSONParser()
	entries, errs := p.Parse(r(input))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
}

func TestJSONParser_InvalidJSON_ProducesError(t *testing.T) {
	p := NewJSONParser()
	entries, errs := p.Parse(r("not json at all"))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(gotErrs), gotErrs)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(got))
	}
}

func TestJSONParser_ErrorMessageContainsLineNumber(t *testing.T) {
	input := `{"valid":"yes"}
not json`
	p := NewJSONParser()
	entries, errs := p.Parse(r(input))
	_, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(gotErrs))
	}
	if !strings.Contains(gotErrs[0].Error(), "line 2") {
		t.Errorf("expected error to contain line number, got: %v", gotErrs[0])
	}
}

func TestJSONParser_MixedValidAndInvalid(t *testing.T) {
	input := `{"level":"info","msg":"valid"}
not json
{"level":"error","msg":"also valid"}`
	p := NewJSONParser()
	entries, errs := p.Parse(r(input))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(gotErrs))
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
}

func TestJSONParser_NumericValues_BecomeFloat64(t *testing.T) {
	p := NewJSONParser()
	entries, errs := p.Parse(r(`{"count":42,"ratio":3.14}`))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0]["count"] != float64(42) {
		t.Errorf("count: got %v (%T), want float64(42)", got[0]["count"], got[0]["count"])
	}
	if got[0]["ratio"] != float64(3.14) {
		t.Errorf("ratio: got %v, want 3.14", got[0]["ratio"])
	}
}

func TestJSONParser_BooleanValues_Preserved(t *testing.T) {
	p := NewJSONParser()
	entries, errs := p.Parse(r(`{"ok":true,"fail":false}`))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if got[0]["ok"] != true {
		t.Errorf("ok: got %v, want true", got[0]["ok"])
	}
	if got[0]["fail"] != false {
		t.Errorf("fail: got %v, want false", got[0]["fail"])
	}
}

func TestJSONParser_NullValue_Preserved(t *testing.T) {
	p := NewJSONParser()
	entries, errs := p.Parse(r(`{"key":null}`))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0]["key"] != nil {
		t.Errorf("key: got %v, want nil", got[0]["key"])
	}
}

func TestJSONParser_NestedObject_Preserved(t *testing.T) {
	p := NewJSONParser()
	entries, errs := p.Parse(r(`{"level":"info","meta":{"host":"srv1"}}`))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if _, ok := got[0]["meta"]; !ok {
		t.Error("expected meta field to be present")
	}
}

func TestJSONParser_AllFieldsPreserved(t *testing.T) {
	p := NewJSONParser()
	entries, errs := p.Parse(r(`{"a":"1","b":"2","c":"3"}`))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got[0]) != 3 {
		t.Errorf("expected 3 fields, got %d: %v", len(got[0]), got[0])
	}
}

func TestJSONParser_MultipleErrors_AllReported(t *testing.T) {
	input := "bad1\nbad2\nbad3"
	p := NewJSONParser()
	entries, errs := p.Parse(r(input))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(got) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(got))
	}
	// The errors channel has buffer 1, but our collectEntries consumer drains it
	// concurrently so the producer goroutine is never permanently blocked; all
	// three errors must be delivered.
	if len(gotErrs) != 3 {
		t.Fatalf("expected 3 errors, got %d: %v", len(gotErrs), gotErrs)
	}
}

// =============================================================================
// LogfmtParser
// =============================================================================

func TestLogfmtParser_EmptyReader(t *testing.T) {
	p := NewLogfmtParser()
	entries, errs := p.Parse(r(""))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(got))
	}
}

func TestLogfmtParser_SkipsEmptyLines(t *testing.T) {
	input := "level=info\n\nlevel=error\n"
	p := NewLogfmtParser()
	entries, errs := p.Parse(r(input))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
}

func TestLogfmtParser_SingleKeyValueLine(t *testing.T) {
	p := NewLogfmtParser()
	entries, errs := p.Parse(r("level=info"))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0]["level"] != "info" {
		t.Errorf("level: got %v, want info", got[0]["level"])
	}
}

func TestLogfmtParser_MultipleKeyValuesOnOneLine(t *testing.T) {
	p := NewLogfmtParser()
	entries, errs := p.Parse(r("level=info msg=hello"))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0]["level"] != "info" {
		t.Errorf("level: got %v, want info", got[0]["level"])
	}
	if got[0]["msg"] != "hello" {
		t.Errorf("msg: got %v, want hello", got[0]["msg"])
	}
}

func TestLogfmtParser_MultipleLines(t *testing.T) {
	input := "level=info msg=first\nlevel=error msg=second\n"
	p := NewLogfmtParser()
	entries, errs := p.Parse(r(input))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0]["level"] != "info" {
		t.Errorf("entry 0 level: got %v, want info", got[0]["level"])
	}
	if got[1]["level"] != "error" {
		t.Errorf("entry 1 level: got %v, want error", got[1]["level"])
	}
}

func TestLogfmtParser_BooleanFlagLine(t *testing.T) {
	p := NewLogfmtParser()
	entries, errs := p.Parse(r("verbose"))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0]["verbose"] != true {
		t.Errorf("verbose: got %v, want true", got[0]["verbose"])
	}
}

func TestLogfmtParser_UnterminatedString_ProducesError(t *testing.T) {
	p := NewLogfmtParser()
	entries, errs := p.Parse(r(`level="unterminated`))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(got) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(got))
	}
	if len(gotErrs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(gotErrs), gotErrs)
	}
}

func TestLogfmtParser_QuotedValue(t *testing.T) {
	p := NewLogfmtParser()
	entries, errs := p.Parse(r(`msg="hello world" level=info`))
	got, gotErrs := collectEntries(t, entries, errs)

	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %v", gotErrs)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0]["msg"] != "hello world" {
		t.Errorf("msg: got %v, want \"hello world\"", got[0]["msg"])
	}
	if got[0]["level"] != "info" {
		t.Errorf("level: got %v, want info", got[0]["level"])
	}
}

// =============================================================================
// parseLogfmt (white-box: same package)
// =============================================================================

func TestParseLogfmt_EmptyString(t *testing.T) {
	entry, err := parseLogfmt("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entry) != 0 {
		t.Errorf("expected empty entry, got %v", entry)
	}
}

func TestParseLogfmt_WhitespaceOnly(t *testing.T) {
	entry, err := parseLogfmt("   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entry) != 0 {
		t.Errorf("expected empty entry, got %v", entry)
	}
}

func TestParseLogfmt_BooleanFlag_NoEquals(t *testing.T) {
	entry, err := parseLogfmt("verbose")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry["verbose"] != true {
		t.Errorf("verbose: got %v, want true", entry["verbose"])
	}
}

func TestParseLogfmt_BooleanFlag_StoresEntireRemaining(t *testing.T) {
	// When there is no '=' anywhere in the line the whole trimmed string
	// is stored as a boolean flag (eqIdx == -1 → entry[remaining] = true; break).
	entry, err := parseLogfmt("verbose debug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry["verbose debug"] != true {
		t.Errorf("expected entry[\"verbose debug\"]=true, got: %v", entry)
	}
}

func TestParseLogfmt_SingleKeyValue(t *testing.T) {
	entry, err := parseLogfmt("key=value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry["key"] != "value" {
		t.Errorf("key: got %v, want value", entry["key"])
	}
}

func TestParseLogfmt_MultipleKeyValues(t *testing.T) {
	entry, err := parseLogfmt("a=1 b=2 c=3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entry) != 3 {
		t.Errorf("expected 3 fields, got %d: %v", len(entry), entry)
	}
	if entry["a"] != "1" {
		t.Errorf("a: got %v, want 1", entry["a"])
	}
	if entry["b"] != "2" {
		t.Errorf("b: got %v, want 2", entry["b"])
	}
	if entry["c"] != "3" {
		t.Errorf("c: got %v, want 3", entry["c"])
	}
}

func TestParseLogfmt_QuotedValue(t *testing.T) {
	entry, err := parseLogfmt(`msg="hello world" level=info`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry["msg"] != "hello world" {
		t.Errorf("msg: got %v, want \"hello world\"", entry["msg"])
	}
	if entry["level"] != "info" {
		t.Errorf("level: got %v, want info", entry["level"])
	}
}

func TestParseLogfmt_QuotedValueOnly(t *testing.T) {
	entry, err := parseLogfmt(`msg="just quoted"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry["msg"] != "just quoted" {
		t.Errorf("msg: got %v, want \"just quoted\"", entry["msg"])
	}
}

func TestParseLogfmt_UnterminatedString_ReturnsError(t *testing.T) {
	_, err := parseLogfmt(`msg="unterminated`)
	if err == nil {
		t.Error("expected error for unterminated string value, got nil")
	}
}

func TestParseLogfmt_QuotedValueWithEscapedQuote(t *testing.T) {
	// The parser skips over `\"` inside a quoted value (endIdx-1 check).
	entry, err := parseLogfmt(`msg="say \"hello\""`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := entry["msg"]; !exists {
		t.Error("expected msg field to be present")
	}
	// The raw bytes between the outer quotes are stored as-is.
	if entry["msg"] != `say \"hello\"` {
		t.Errorf("msg: got %v, want %q", entry["msg"], `say \"hello\"`)
	}
}

func TestParseLogfmt_LeadingAndTrailingSpaces(t *testing.T) {
	entry, err := parseLogfmt("  level=info  msg=hello  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry["level"] != "info" {
		t.Errorf("level: got %v, want info", entry["level"])
	}
	if entry["msg"] != "hello" {
		t.Errorf("msg: got %v, want hello", entry["msg"])
	}
}

func TestParseLogfmt_EmptyValue(t *testing.T) {
	// "key=" — value is empty string (no chars before next space or end).
	entry, err := parseLogfmt("key=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry["key"] != "" {
		t.Errorf("key: got %v, want empty string", entry["key"])
	}
}
