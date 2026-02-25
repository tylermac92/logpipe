// Package filter provides log entry filtering based on field values.
// Filters are composed from simple field expressions and can be combined
// into a composite AND filter.
package filter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tylermac92/logpipe/internal/parser"
)

// Filter is the interface implemented by all log entry filters.
// Match returns true when the given entry satisfies the filter condition.
type Filter interface {
	Match(entry parser.LogEntry) bool
}

// FieldFilter matches log entries by comparing a named field against a
// constant value using a specific operator.
type FieldFilter struct {
	re       *regexp.Regexp // Compiled regex, populated only for the ~ operator.
	Field    string         // Name of the log field to inspect.
	Operator string         // Comparison operator (=, !=, >, <, >=, <=, ~).
	Value    string         // The value to compare against.
}

// NewFieldFilter parses a filter expression of the form "field<op>value" and
// returns a FieldFilter. Supported operators, in precedence order:
//
//	!=   not equal
//	~    regex match
//	>=   greater-than-or-equal (lexicographic)
//	<=   less-than-or-equal (lexicographic)
//	=    equal
//	>    greater-than (lexicographic)
//	<    less-than (lexicographic)
//
// Returns an error if the expression contains no recognised operator or if
// the ~ operator is paired with an invalid regular expression.
func NewFieldFilter(expression string) (*FieldFilter, error) {
	// Operators are checked in this order so that multi-character operators
	// (e.g. "!=", ">=") are matched before their single-character prefixes.
	operators := []string{"!=", "~", ">=", "<=", "=", ">", "<"}

	for _, op := range operators {
		idx := strings.Index(expression, op)
		if idx == -1 {
			continue
		}

		field := expression[:idx]
		value := expression[idx+len(op):]
		f := &FieldFilter{
			Field:    field,
			Operator: op,
			Value:    value,
		}

		if op == "~" {
			re, err := regexp.Compile(value)
			if err != nil {
				return nil, fmt.Errorf("invalid regex in filter: %w", err)
			}
			f.re = re
		}

		return f, nil
	}

	return nil, fmt.Errorf("invalid filter expression: %s", expression)
}

// Match returns true when the entry's field satisfies the filter condition.
// The field value is converted to a string via fmt.Sprintf before comparison,
// so numeric and boolean field values are supported. Entries that do not
// contain the target field always return false.
func (f *FieldFilter) Match(entry parser.LogEntry) bool {
	value, exists := entry[f.Field]
	if !exists {
		return false
	}

	switch f.Operator {
	case "=":
		return fmt.Sprintf("%v", value) == f.Value
	case "!=":
		return fmt.Sprintf("%v", value) != f.Value
	case ">":
		return fmt.Sprintf("%v", value) > f.Value
	case "<":
		return fmt.Sprintf("%v", value) < f.Value
	case ">=":
		return fmt.Sprintf("%v", value) >= f.Value
	case "<=":
		return fmt.Sprintf("%v", value) <= f.Value
	case "~":
		return f.re.MatchString(fmt.Sprintf("%v", value))
	default:
		return false
	}
}

// CompositeFilter combines multiple filters with logical AND semantics:
// an entry must satisfy every child filter to be considered a match.
type CompositeFilter struct {
	filters []Filter
}

// NewCompositeFilter returns a CompositeFilter that requires all provided
// filters to match. Passing zero filters creates a filter that matches
// every entry.
func NewCompositeFilter(filters ...Filter) *CompositeFilter {
	return &CompositeFilter{filters: filters}
}

// Match returns true only if every child filter matches the entry.
// An empty CompositeFilter always returns true.
func (cf *CompositeFilter) Match(entry parser.LogEntry) bool {
	for _, filter := range cf.filters {
		if !filter.Match(entry) {
			return false
		}
	}
	return true
}
