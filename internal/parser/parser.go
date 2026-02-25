// Package parser provides log entry parsers for different log formats.
// Parsers read from an io.Reader and emit log entries over a channel,
// reporting parse errors on a separate error channel.
package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// LogEntry represents a single structured log record as a map of field names to values.
type LogEntry map[string]any

// Parser is the interface implemented by all log format parsers.
// Parse reads from r and returns two channels: one for successfully parsed
// log entries and one for errors encountered during parsing. Both channels
// are closed when r is exhausted.
type Parser interface {
	Parse(r io.Reader) (<-chan LogEntry, <-chan error)
}

// JSONParser parses newline-delimited JSON log entries.
type JSONParser struct{}

// NewJSONParser returns a new JSONParser.
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse reads newline-delimited JSON from r, emitting each successfully
// unmarshalled object as a LogEntry. Lines that fail to parse are sent to
// the error channel and skipped. The scanner buffer is set to 1 MiB to
// handle unusually long log lines.
func (p *JSONParser) Parse(r io.Reader) (<-chan LogEntry, <-chan error) {
	entries := make(chan LogEntry)
	errors := make(chan error, 1)

	go func() {
		defer close(entries)
		defer close(errors)

		scanner := bufio.NewScanner(r)
		// Increase the scanner buffer to accommodate large JSON log lines.
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var entry LogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				errors <- fmt.Errorf("line %d: %w", lineNum, err)
				continue
			}

			entries <- entry
		}

		if err := scanner.Err(); err != nil {
			errors <- fmt.Errorf("scanner error: %w", err)
		}
	}()

	return entries, errors
}

// LogfmtParser parses logfmt-formatted log entries.
// Logfmt is a simple key=value format popularized by Heroku and the Go
// ecosystem (e.g. github.com/kr/logfmt).
type LogfmtParser struct{}

// NewLogfmtParser returns a new LogfmtParser.
func NewLogfmtParser() *LogfmtParser {
	return &LogfmtParser{}
}

// Parse reads logfmt lines from r, emitting each successfully parsed line
// as a LogEntry. Lines that fail to parse are sent to the error channel
// and skipped.
func (p *LogfmtParser) Parse(r io.Reader) (<-chan LogEntry, <-chan error) {
	entries := make(chan LogEntry)
	errors := make(chan error, 1)

	go func() {
		defer close(entries)
		defer close(errors)

		scanner := bufio.NewScanner(r)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			entry, err := parseLogfmt(line)
			if err != nil {
				errors <- fmt.Errorf("line %d: %w", lineNum, err)
				continue
			}

			entries <- entry
		}
	}()

	return entries, errors
}

// parseLogfmt parses a single logfmt line into a LogEntry.
//
// The logfmt format consists of space-separated key=value pairs. Values may
// be unquoted tokens or double-quoted strings (with backslash escaping).
// A bare key with no '=' is stored with a boolean true value.
func parseLogfmt(line string) (LogEntry, error) {
	entry := make(LogEntry)
	remaining := line

	for remaining != "" {
		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			break
		}

		eqIdx := strings.IndexByte(remaining, '=')
		if eqIdx == -1 {
			// Bare key with no value — treat as a boolean flag.
			entry[remaining] = true
			break
		}

		key := remaining[:eqIdx]
		remaining = remaining[eqIdx+1:]

		var value string
		if strings.HasPrefix(remaining, `"`) {
			// Quoted value: scan forward to find the closing unescaped quote.
			endIdx := 1
			for endIdx < len(remaining) {
				if remaining[endIdx] == '"' && remaining[endIdx-1] != '\\' {
					break
				}
				endIdx++
			}
			if endIdx >= len(remaining) {
				return nil, fmt.Errorf("unterminated string value")
			}
			value = remaining[1:endIdx]
			remaining = remaining[endIdx+1:]
		} else {
			// Unquoted value: ends at the next space.
			spaceIdx := strings.IndexByte(remaining, ' ')
			if spaceIdx == -1 {
				value = remaining
				remaining = ""
			} else {
				value = remaining[:spaceIdx]
				remaining = remaining[spaceIdx+1:]
			}
		}
		entry[key] = value
	}
	return entry, nil
}
