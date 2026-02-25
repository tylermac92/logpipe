package filter

import (
	"testing"

	"github.com/tylermac92/logpipe/internal/parser"
)

// =============================================================================
// NewFieldFilter — expression parsing
// =============================================================================

func TestNewFieldFilter_Equal(t *testing.T) {
	f, err := NewFieldFilter("level=error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Field != "level" {
		t.Errorf("field: got %q, want %q", f.Field, "level")
	}
	if f.Operator != "=" {
		t.Errorf("operator: got %q, want %q", f.Operator, "=")
	}
	if f.Value != "error" {
		t.Errorf("value: got %q, want %q", f.Value, "error")
	}
}

func TestNewFieldFilter_NotEqual(t *testing.T) {
	f, err := NewFieldFilter("level!=info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Operator != "!=" {
		t.Errorf("operator: got %q, want %q", f.Operator, "!=")
	}
	if f.Field != "level" {
		t.Errorf("field: got %q, want %q", f.Field, "level")
	}
	if f.Value != "info" {
		t.Errorf("value: got %q, want %q", f.Value, "info")
	}
}

func TestNewFieldFilter_GreaterThanOrEqual(t *testing.T) {
	f, err := NewFieldFilter("count>=10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Operator != ">=" {
		t.Errorf("operator: got %q, want %q", f.Operator, ">=")
	}
	if f.Field != "count" {
		t.Errorf("field: got %q, want %q", f.Field, "count")
	}
	if f.Value != "10" {
		t.Errorf("value: got %q, want %q", f.Value, "10")
	}
}

func TestNewFieldFilter_LessThanOrEqual(t *testing.T) {
	f, err := NewFieldFilter("count<=5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Operator != "<=" {
		t.Errorf("operator: got %q, want %q", f.Operator, "<=")
	}
}

func TestNewFieldFilter_GreaterThan(t *testing.T) {
	f, err := NewFieldFilter("count>5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Operator != ">" {
		t.Errorf("operator: got %q, want %q", f.Operator, ">")
	}
}

func TestNewFieldFilter_LessThan(t *testing.T) {
	f, err := NewFieldFilter("count<5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Operator != "<" {
		t.Errorf("operator: got %q, want %q", f.Operator, "<")
	}
}

func TestNewFieldFilter_Regex(t *testing.T) {
	f, err := NewFieldFilter("msg~error.*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Operator != "~" {
		t.Errorf("operator: got %q, want %q", f.Operator, "~")
	}
	if f.re == nil {
		t.Error("expected compiled regex, got nil")
	}
}

func TestNewFieldFilter_Regex_FieldAndValueParsed(t *testing.T) {
	f, err := NewFieldFilter("msg~^fatal:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Field != "msg" {
		t.Errorf("field: got %q, want %q", f.Field, "msg")
	}
	if f.Value != "^fatal:" {
		t.Errorf("value: got %q, want %q", f.Value, "^fatal:")
	}
}

// The operator scan order is: "!=", "~", ">=", "<=", "=", ">", "<".
// NotEqual must be tried before Equal so "level!=error" is parsed correctly.
func TestNewFieldFilter_NotEqualTakesPriorityOverEqual(t *testing.T) {
	f, err := NewFieldFilter("level!=error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Operator != "!=" {
		t.Errorf("operator: got %q, want %q — NotEqual must take priority over Equal", f.Operator, "!=")
	}
	if f.Field != "level" {
		t.Errorf("field: got %q, want %q", f.Field, "level")
	}
	if f.Value != "error" {
		t.Errorf("value: got %q, want %q", f.Value, "error")
	}
}

// ">=" must be tried before ">" so "count>=10" is not misread as field="count>", value="10".
func TestNewFieldFilter_GreaterThanOrEqualTakesPriorityOverGreaterThan(t *testing.T) {
	f, err := NewFieldFilter("count>=10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Operator != ">=" {
		t.Errorf("operator: got %q, want %q — >= must take priority over >", f.Operator, ">=")
	}
}

func TestNewFieldFilter_InvalidExpression_NoOperator(t *testing.T) {
	_, err := NewFieldFilter("noop")
	if err == nil {
		t.Error("expected error for expression with no recognised operator")
	}
}

func TestNewFieldFilter_InvalidExpression_Empty(t *testing.T) {
	_, err := NewFieldFilter("")
	if err == nil {
		t.Error("expected error for empty expression")
	}
}

func TestNewFieldFilter_InvalidRegex(t *testing.T) {
	_, err := NewFieldFilter("msg~[invalid")
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestNewFieldFilter_ValueWithSpecialChars(t *testing.T) {
	// Field filter value can contain anything after the operator.
	f, err := NewFieldFilter("time>=2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Field != "time" {
		t.Errorf("field: got %q, want %q", f.Field, "time")
	}
	if f.Value != "2024-01-01T00:00:00Z" {
		t.Errorf("value: got %q, want %q", f.Value, "2024-01-01T00:00:00Z")
	}
}

// =============================================================================
// FieldFilter.Match
// =============================================================================

func TestFieldFilter_Match_Equal_Hit(t *testing.T) {
	f, _ := NewFieldFilter("level=error")
	entry := parser.LogEntry{"level": "error"}
	if !f.Match(entry) {
		t.Error("expected Match=true")
	}
}

func TestFieldFilter_Match_Equal_Miss(t *testing.T) {
	f, _ := NewFieldFilter("level=error")
	entry := parser.LogEntry{"level": "info"}
	if f.Match(entry) {
		t.Error("expected Match=false")
	}
}

func TestFieldFilter_Match_Equal_MissingField(t *testing.T) {
	f, _ := NewFieldFilter("level=error")
	entry := parser.LogEntry{"msg": "something"}
	if f.Match(entry) {
		t.Error("expected Match=false for missing field")
	}
}

func TestFieldFilter_Match_Equal_EmptyEntry(t *testing.T) {
	f, _ := NewFieldFilter("level=error")
	if f.Match(parser.LogEntry{}) {
		t.Error("expected Match=false for empty entry")
	}
}

func TestFieldFilter_Match_NotEqual_Hit(t *testing.T) {
	f, _ := NewFieldFilter("level!=info")
	entry := parser.LogEntry{"level": "error"}
	if !f.Match(entry) {
		t.Error("expected Match=true")
	}
}

func TestFieldFilter_Match_NotEqual_Miss(t *testing.T) {
	f, _ := NewFieldFilter("level!=info")
	entry := parser.LogEntry{"level": "info"}
	if f.Match(entry) {
		t.Error("expected Match=false")
	}
}

func TestFieldFilter_Match_NotEqual_MissingField(t *testing.T) {
	f, _ := NewFieldFilter("level!=info")
	// Field does not exist — Match returns false regardless of operator.
	entry := parser.LogEntry{"msg": "hello"}
	if f.Match(entry) {
		t.Error("expected Match=false for missing field")
	}
}

func TestFieldFilter_Match_GreaterThan_Hit(t *testing.T) {
	f, _ := NewFieldFilter("level>b")
	if !f.Match(parser.LogEntry{"level": "c"}) {
		t.Error("expected Match=true")
	}
}

func TestFieldFilter_Match_GreaterThan_Miss(t *testing.T) {
	f, _ := NewFieldFilter("level>c")
	if f.Match(parser.LogEntry{"level": "a"}) {
		t.Error("expected Match=false")
	}
}

func TestFieldFilter_Match_GreaterThan_EqualValue_Miss(t *testing.T) {
	f, _ := NewFieldFilter("level>a")
	if f.Match(parser.LogEntry{"level": "a"}) {
		t.Error("expected Match=false when values are equal")
	}
}

func TestFieldFilter_Match_LessThan_Hit(t *testing.T) {
	f, _ := NewFieldFilter("level<c")
	if !f.Match(parser.LogEntry{"level": "a"}) {
		t.Error("expected Match=true")
	}
}

func TestFieldFilter_Match_LessThan_Miss(t *testing.T) {
	f, _ := NewFieldFilter("level<a")
	if f.Match(parser.LogEntry{"level": "c"}) {
		t.Error("expected Match=false")
	}
}

func TestFieldFilter_Match_LessThan_EqualValue_Miss(t *testing.T) {
	f, _ := NewFieldFilter("level<a")
	if f.Match(parser.LogEntry{"level": "a"}) {
		t.Error("expected Match=false when values are equal")
	}
}

func TestFieldFilter_Match_GreaterThanOrEqual_Equal_Hit(t *testing.T) {
	f, _ := NewFieldFilter("level>=b")
	if !f.Match(parser.LogEntry{"level": "b"}) {
		t.Error("expected Match=true for equal value")
	}
}

func TestFieldFilter_Match_GreaterThanOrEqual_Greater_Hit(t *testing.T) {
	f, _ := NewFieldFilter("level>=b")
	if !f.Match(parser.LogEntry{"level": "c"}) {
		t.Error("expected Match=true for greater value")
	}
}

func TestFieldFilter_Match_GreaterThanOrEqual_Less_Miss(t *testing.T) {
	f, _ := NewFieldFilter("level>=c")
	if f.Match(parser.LogEntry{"level": "a"}) {
		t.Error("expected Match=false for lesser value")
	}
}

func TestFieldFilter_Match_LessThanOrEqual_Equal_Hit(t *testing.T) {
	f, _ := NewFieldFilter("level<=b")
	if !f.Match(parser.LogEntry{"level": "b"}) {
		t.Error("expected Match=true for equal value")
	}
}

func TestFieldFilter_Match_LessThanOrEqual_Less_Hit(t *testing.T) {
	f, _ := NewFieldFilter("level<=b")
	if !f.Match(parser.LogEntry{"level": "a"}) {
		t.Error("expected Match=true for lesser value")
	}
}

func TestFieldFilter_Match_LessThanOrEqual_Greater_Miss(t *testing.T) {
	f, _ := NewFieldFilter("level<=a")
	if f.Match(parser.LogEntry{"level": "c"}) {
		t.Error("expected Match=false for greater value")
	}
}

func TestFieldFilter_Match_Regex_Hit(t *testing.T) {
	f, _ := NewFieldFilter("msg~^err.*")
	if !f.Match(parser.LogEntry{"msg": "error: connection refused"}) {
		t.Error("expected Match=true for matching regex")
	}
}

func TestFieldFilter_Match_Regex_Miss(t *testing.T) {
	f, _ := NewFieldFilter("msg~^err.*")
	if f.Match(parser.LogEntry{"msg": "info: all systems go"}) {
		t.Error("expected Match=false for non-matching regex")
	}
}

func TestFieldFilter_Match_Regex_CaseSensitive(t *testing.T) {
	f, _ := NewFieldFilter("msg~^ERROR")
	if f.Match(parser.LogEntry{"msg": "error: lowercase"}) {
		t.Error("expected Match=false: regex is case-sensitive by default")
	}
}

func TestFieldFilter_Match_Regex_Partial(t *testing.T) {
	f, _ := NewFieldFilter("msg~timeout")
	if !f.Match(parser.LogEntry{"msg": "connection timeout exceeded"}) {
		t.Error("expected Match=true for partial regex match")
	}
}

// The Match function calls fmt.Sprintf("%v", value) for comparison.
// JSON numbers arrive as float64, so float64(42) formats as "42".
func TestFieldFilter_Match_NumericField_StringComparison(t *testing.T) {
	f, _ := NewFieldFilter("count=42")
	if !f.Match(parser.LogEntry{"count": float64(42)}) {
		t.Error("expected Match=true: float64(42) → \"42\"")
	}
}

func TestFieldFilter_Match_BooleanField_StringComparison(t *testing.T) {
	f, _ := NewFieldFilter("ok=true")
	if !f.Match(parser.LogEntry{"ok": true}) {
		t.Error("expected Match=true: true → \"true\"")
	}
}

func TestFieldFilter_Match_NilField_StringComparison(t *testing.T) {
	f, _ := NewFieldFilter("key=<nil>")
	if !f.Match(parser.LogEntry{"key": nil}) {
		t.Error("expected Match=true: nil → \"<nil>\"")
	}
}

// =============================================================================
// CompositeFilter
// =============================================================================

func TestCompositeFilter_NoFilters_MatchesAnyEntry(t *testing.T) {
	cf := NewCompositeFilter()
	entry := parser.LogEntry{"level": "info", "msg": "anything"}
	if !cf.Match(entry) {
		t.Error("expected empty CompositeFilter to match any entry")
	}
}

func TestCompositeFilter_NoFilters_MatchesEmptyEntry(t *testing.T) {
	cf := NewCompositeFilter()
	if !cf.Match(parser.LogEntry{}) {
		t.Error("expected empty CompositeFilter to match empty entry")
	}
}

func TestCompositeFilter_SingleFilter_Hit(t *testing.T) {
	f, _ := NewFieldFilter("level=info")
	cf := NewCompositeFilter(f)
	if !cf.Match(parser.LogEntry{"level": "info"}) {
		t.Error("expected Match=true")
	}
}

func TestCompositeFilter_SingleFilter_Miss(t *testing.T) {
	f, _ := NewFieldFilter("level=info")
	cf := NewCompositeFilter(f)
	if cf.Match(parser.LogEntry{"level": "error"}) {
		t.Error("expected Match=false")
	}
}

func TestCompositeFilter_TwoFilters_BothMatch(t *testing.T) {
	f1, _ := NewFieldFilter("level=error")
	f2, _ := NewFieldFilter("service=api")
	cf := NewCompositeFilter(f1, f2)
	entry := parser.LogEntry{"level": "error", "service": "api"}
	if !cf.Match(entry) {
		t.Error("expected Match=true when all filters match")
	}
}

func TestCompositeFilter_TwoFilters_FirstMisses(t *testing.T) {
	f1, _ := NewFieldFilter("level=error")
	f2, _ := NewFieldFilter("service=api")
	cf := NewCompositeFilter(f1, f2)
	entry := parser.LogEntry{"level": "info", "service": "api"}
	if cf.Match(entry) {
		t.Error("expected Match=false (first filter misses)")
	}
}

func TestCompositeFilter_TwoFilters_SecondMisses(t *testing.T) {
	f1, _ := NewFieldFilter("level=error")
	f2, _ := NewFieldFilter("service=api")
	cf := NewCompositeFilter(f1, f2)
	entry := parser.LogEntry{"level": "error", "service": "web"}
	if cf.Match(entry) {
		t.Error("expected Match=false (second filter misses)")
	}
}

func TestCompositeFilter_TwoFilters_NoneMatch(t *testing.T) {
	f1, _ := NewFieldFilter("level=error")
	f2, _ := NewFieldFilter("service=api")
	cf := NewCompositeFilter(f1, f2)
	entry := parser.LogEntry{"level": "info", "service": "web"}
	if cf.Match(entry) {
		t.Error("expected Match=false (no filters match)")
	}
}

func TestCompositeFilter_ThreeFilters_AllMatch(t *testing.T) {
	f1, _ := NewFieldFilter("level=error")
	f2, _ := NewFieldFilter("service=api")
	f3, _ := NewFieldFilter("region=us-east")
	cf := NewCompositeFilter(f1, f2, f3)
	entry := parser.LogEntry{"level": "error", "service": "api", "region": "us-east"}
	if !cf.Match(entry) {
		t.Error("expected Match=true when all three filters match")
	}
}

func TestCompositeFilter_ThreeFilters_MiddleMisses(t *testing.T) {
	f1, _ := NewFieldFilter("level=error")
	f2, _ := NewFieldFilter("service=api")
	f3, _ := NewFieldFilter("region=us-east")
	cf := NewCompositeFilter(f1, f2, f3)
	entry := parser.LogEntry{"level": "error", "service": "web", "region": "us-east"}
	if cf.Match(entry) {
		t.Error("expected Match=false (middle filter misses)")
	}
}

// A CompositeFilter with a regex filter and an equality filter ANDs both conditions.
func TestCompositeFilter_MixedOperators(t *testing.T) {
	fEq, _ := NewFieldFilter("level=error")
	fRe, _ := NewFieldFilter("msg~timeout")
	cf := NewCompositeFilter(fEq, fRe)

	// Both conditions satisfied.
	if !cf.Match(parser.LogEntry{"level": "error", "msg": "db timeout"}) {
		t.Error("expected Match=true when both conditions hold")
	}
	// Regex not satisfied.
	if cf.Match(parser.LogEntry{"level": "error", "msg": "normal error"}) {
		t.Error("expected Match=false when regex does not match")
	}
}
