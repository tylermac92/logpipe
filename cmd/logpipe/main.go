// Command logpipe reads structured log entries from a file or stdin,
// optionally filters them, and writes them to stdout in the requested format.
//
// Usage:
//
//	logpipe [flags]
//
// See the README or run with -help for a full flag reference.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tylermac92/logpipe/internal/filter"
	"github.com/tylermac92/logpipe/internal/formatter"
	"github.com/tylermac92/logpipe/internal/parser"
)

// multiFlag is a custom flag.Value that accumulates repeated uses of the same
// flag into a string slice. It is used so that -filter can be specified more
// than once on the command line.
type multiFlag []string

// String implements flag.Value and returns a comma-joined representation of
// all collected values.
func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

// Set implements flag.Value and appends value to the slice.
func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	// --- Flag definitions ---
	var (
		format      = flag.String("format", "text", "Output format: text or json")
		inputFormat = flag.String("input", "json", "Input format: json, logfmt")
		filePath    = flag.String("file", "", "Path to log file (default: stdin)")
		color       = flag.Bool("color", false, "Enable color output (text format only)")
		pretty      = flag.Bool("pretty", false, "Pretty-print JSON output (json format only)")
		fields      = flag.String("fields", "", "Comma-separated list of fields to display (text format)")
		filters     multiFlag
	)

	flag.Var(&filters, "filter", "Filter expression (e.g. level=error, time>=2024-01-01T00:00:00Z)")
	flag.Parse()

	// --- Input source ---
	// Open the specified file, or fall back to stdin.
	var r io.Reader
	if *filePath != "" {
		f, err := os.Open(*filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		r = f
	} else {
		r = os.Stdin
	}

	// --- Parser selection ---
	var p parser.Parser
	switch *inputFormat {
	case "json":
		p = parser.NewJSONParser()
	case "logfmt":
		p = parser.NewLogfmtParser()
	default:
		fmt.Fprintf(os.Stderr, "Unsupported input format: %s\n", *inputFormat)
		os.Exit(1)
	}

	// --- Filter construction ---
	// Parse each -filter flag into a FieldFilter and combine them with AND
	// semantics using a CompositeFilter.
	var filterList []filter.Filter
	for _, f := range filters {
		filt, err := filter.NewFieldFilter(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid filter: %v\n", err)
			os.Exit(1)
		}
		filterList = append(filterList, filt)
	}
	composite := filter.NewCompositeFilter(filterList...)

	// --- Formatter selection ---
	var fieldsList []string
	if *fields != "" {
		fieldsList = strings.Split(*fields, ",")
	}

	var fmt_ formatter.Formatter
	switch *format {
	case "json":
		fmt_ = &formatter.JSONFormatter{Pretty: *pretty}
	case "text":
		fmt_ = &formatter.TextFormatter{Color: *color, Fields: fieldsList}
	case "logfmt":
		fmt_ = &formatter.LogfmtFormatter{}
	default:
		fmt.Fprintf(os.Stderr, "Unsupported output format: %s\n", *format)
		os.Exit(1)
	}

	// --- Processing pipeline ---
	// Parse entries and errors from concurrent goroutines inside the parser.
	entries, errs := p.Parse(r)

	// Drain parse errors asynchronously so they don't block the entry channel.
	go func() {
		for err := range errs {
			fmt.Fprintf(os.Stderr, "Error parsing log: %v\n", err)
		}
	}()

	// Iterate over parsed entries, apply filters, and format matching ones.
	exitCode := 0
	for entry := range entries {
		if composite.Match(entry) {
			if err := fmt_.Format(os.Stdout, entry); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting log: %v\n", err)
				exitCode = 1
			}
		}
	}

	os.Exit(exitCode)
}
