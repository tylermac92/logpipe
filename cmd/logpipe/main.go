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
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tylermac92/logpipe/internal/filter"
	"github.com/tylermac92/logpipe/internal/formatter"
	"github.com/tylermac92/logpipe/internal/parser"
)

// mergedEntry pairs a parsed log entry with its timestamp for sorting and the
// source file name already embedded in the entry under the "_source" key.
type mergedEntry struct {
	entry parser.LogEntry
	t     time.Time // zero when no recognisable timestamp field is present
}

// parseTimestampForSort extracts and parses a timestamp from entry for
// comparison purposes. It checks the canonical timestamp field names in order
// and tries a Unix-float and then RFC 3339 interpretation. Returns the zero
// time when no usable timestamp is found.
func parseTimestampForSort(entry parser.LogEntry) time.Time {
	for _, key := range []string{"time", "ts", "timestamp"} {
		val, ok := entry[key]
		if !ok {
			continue
		}
		s := fmt.Sprintf("%v", val)
		var f float64
		if _, err := fmt.Sscanf(s, "%f", &f); err == nil && f > 1e9 {
			return time.Unix(int64(f), 0).UTC()
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// loadEntries drains all log entries produced by p reading from r, tags each
// entry with _source = source, and returns a slice of mergedEntry ready for
// sorting. Parse errors are printed to stderr and skipped.
func loadEntries(r io.Reader, p parser.Parser, source string) []mergedEntry {
	entries, errs := p.Parse(r)
	go func() {
		for err := range errs {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", source, err)
		}
	}()
	var result []mergedEntry
	for entry := range entries {
		entry["_source"] = source
		result = append(result, mergedEntry{
			entry: entry,
			t:     parseTimestampForSort(entry),
		})
	}
	return result
}

// statEntry holds a single row in the --stats frequency table.
type statEntry struct {
	Value string
	Count int
}

// collectStats drains the entries channel, applies match to each entry, and
// tallies the string representation of the named field's value. Entries that
// do not contain the field are counted under "(none)". The returned slice is
// sorted by count descending; ties are broken alphabetically by value.
func collectStats(entries <-chan parser.LogEntry, match func(parser.LogEntry) bool, field string) []statEntry {
	counts := make(map[string]int)
	for entry := range entries {
		if match(entry) {
			key := "(none)"
			if v, ok := entry[field]; ok {
				key = fmt.Sprintf("%v", v)
			}
			counts[key]++
		}
	}
	result := make([]statEntry, 0, len(counts))
	for v, n := range counts {
		result = append(result, statEntry{v, n})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Value < result[j].Value
	})
	return result
}

// sniffFormat reads the first non-empty line from r to decide whether the
// input is newline-delimited JSON ("json") or logfmt ("logfmt"). It returns
// the detected format name and a reconstructed io.Reader that still contains
// the peeked line so the chosen parser receives the complete byte stream.
// If the input is empty or only whitespace it defaults to "json".
func sniffFormat(r io.Reader) (string, io.Reader, error) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			reconstructed := io.MultiReader(strings.NewReader(line), br)
			if strings.HasPrefix(trimmed, "{") {
				return "json", reconstructed, nil
			}
			return "logfmt", reconstructed, nil
		}
		if err == io.EOF {
			return "json", br, nil
		}
		if err != nil {
			return "", nil, fmt.Errorf("auto-detecting input format: %w", err)
		}
	}
}

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
	var version = "dev"

	// --- Flag definitions ---
	var (
		format      = flag.String("format", "text", "Output format: text or json")
		inputFormat = flag.String("input", "auto", "Input format: json, logfmt, auto (default: auto)")
		filePath    = flag.String("file", "", "Path to log file (default: stdin)")
		color       = flag.Bool("color", false, "Enable color output (text format only)")
		pretty      = flag.Bool("pretty", false, "Pretty-print JSON output (json format only)")
		fields      = flag.String("fields", "", "Comma-separated list of fields to display (text format)")
		filters     multiFlag
		statsField  = flag.String("stats", "", "Print a frequency table of values for the named field instead of formatting entries")
		versionFlag = flag.Bool("version", false, "Print version and exit")
	)

	var mergeFiles multiFlag
	flag.Var(&filters, "filter", "Filter expression (e.g. level=error, time>=2024-01-01T00:00:00Z)")
	flag.Var(&mergeFiles, "merge", "File to include in merged timestamp-sorted output (repeatable; use --merge once per file)")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("logpipe %s\n", version)
		os.Exit(0)
	}

	if *filePath != "" && len(mergeFiles) > 0 {
		fmt.Fprintf(os.Stderr, "--file and --merge are mutually exclusive\n")
		os.Exit(1)
	}

	// --- Input source and parser (single-file / stdin mode only) ---
	var r io.Reader
	var p parser.Parser
	if len(mergeFiles) == 0 {
		// Open the specified file, or fall back to stdin.
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

		switch *inputFormat {
		case "json":
			p = parser.NewJSONParser()
		case "logfmt":
			p = parser.NewLogfmtParser()
		case "auto":
			detected, sniffed, err := sniffFormat(r)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error detecting input format: %v\n", err)
				os.Exit(1)
			}
			r = sniffed
			if detected == "json" {
				p = parser.NewJSONParser()
			} else {
				p = parser.NewLogfmtParser()
			}
		default:
			fmt.Fprintf(os.Stderr, "Unsupported input format: %s\n", *inputFormat)
			os.Exit(1)
		}
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

	// --- Merge pipeline ---
	// When --merge is used, load all files, sort by timestamp, then feed into
	// the same stats / format machinery as the normal pipeline.
	if len(mergeFiles) > 0 {
		var all []mergedEntry
		for _, path := range mergeFiles {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", path, err)
				os.Exit(1)
			}
			defer f.Close()
			detected, sniffed, err := sniffFormat(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error detecting format of %s: %v\n", path, err)
				os.Exit(1)
			}
			var mp parser.Parser
			if detected == "json" {
				mp = parser.NewJSONParser()
			} else {
				mp = parser.NewLogfmtParser()
			}
			all = append(all, loadEntries(sniffed, mp, filepath.Base(path))...)
		}
		sort.SliceStable(all, func(i, j int) bool {
			return all[i].t.Before(all[j].t)
		})

		ch := make(chan parser.LogEntry, len(all))
		for _, me := range all {
			ch <- me.entry
		}
		close(ch)

		if *statsField != "" {
			for _, s := range collectStats(ch, composite.Match, *statsField) {
				fmt.Fprintf(os.Stdout, "%s: %d\n", s.Value, s.Count)
			}
			os.Exit(0)
		}
		exitCode := 0
		for entry := range ch {
			if composite.Match(entry) {
				if err := fmt_.Format(os.Stdout, entry); err != nil {
					fmt.Fprintf(os.Stderr, "Error formatting log: %v\n", err)
					exitCode = 1
				}
			}
		}
		os.Exit(exitCode)
	}

	// --- Normal pipeline ---
	// Parse entries and errors from concurrent goroutines inside the parser.
	entries, errs := p.Parse(r)

	// Drain parse errors asynchronously so they don't block the entry channel.
	go func() {
		for err := range errs {
			fmt.Fprintf(os.Stderr, "Error parsing log: %v\n", err)
		}
	}()

	if *statsField != "" {
		// Stats mode: count value frequencies for the named field and print a
		// frequency table sorted by count descending.
		for _, s := range collectStats(entries, composite.Match, *statsField) {
			fmt.Fprintf(os.Stdout, "%s: %d\n", s.Value, s.Count)
		}
		os.Exit(0)
	}

	// Normal mode: iterate over parsed entries, apply filters, and format matching ones.
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
