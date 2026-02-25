package main

import (
	"testing"
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
