# logpipe

A command-line tool for parsing, filtering, and reformatting structured log streams. It reads newline-delimited log entries from a file or stdin and writes matching entries to stdout in your chosen output format.

## Features

- **Input formats:** JSON (newline-delimited), logfmt
- **Output formats:** human-readable text, JSON, logfmt
- **Filtering:** field-based expressions with `=`, `!=`, `>`, `<`, `>=`, `<=`, and `~` (regex) operators, combined with AND logic
- **Color output:** ANSI-colored level badges for terminal use
- **Field selection:** restrict text output to a specific list of fields
- **Streaming:** processes large log files line-by-line with no buffering of the full file

## Installation

```bash
go install github.com/tylermac92/logpipe/cmd/logpipe@latest
```

Or build from source:

```bash
git clone https://github.com/tylermac92/logpipe.git
cd logpipe
go build -o logpipe ./cmd/logpipe
```

## Usage

```
logpipe [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-input` | `json` | Input format: `json` or `logfmt` |
| `-format` | `text` | Output format: `text`, `json`, or `logfmt` |
| `-file` | *(stdin)* | Path to a log file; omit to read from stdin |
| `-filter` | | Filter expression; may be repeated for AND logic |
| `-fields` | *(all)* | Comma-separated field names to include in `text` output |
| `-color` | `false` | Enable ANSI color in `text` output |
| `-pretty` | `false` | Indent `json` output |

### Filter expressions

A filter expression has the form `field<op>value`.

| Operator | Meaning |
|----------|---------|
| `=` | field equals value |
| `!=` | field does not equal value |
| `>` | field is lexicographically greater than value |
| `<` | field is lexicographically less than value |
| `>=` | field is greater than or equal to value |
| `<=` | field is less than or equal to value |
| `~` | field matches the regular expression `value` |

Multiple `-filter` flags are combined with AND: an entry must satisfy all of them to be printed.

## Examples

**Tail a JSON log file and display it in readable text with color:**
```bash
tail -f /var/log/app.log | logpipe -color
```

**Show only error-level entries:**
```bash
logpipe -file app.log -filter level=error
```

**Filter by level and a time range, output as pretty JSON:**
```bash
logpipe -file app.log \
  -filter level=error \
  -filter "time>=2024-01-15T00:00:00Z" \
  -format json -pretty
```

**Parse logfmt input and display specific fields:**
```bash
logpipe -input logfmt -fields time,level,msg,request_id -file app.log
```

**Regex filter — find entries whose message contains "timeout":**
```bash
logpipe -file app.log -filter "msg~timeout"
```

**Convert a JSON log to logfmt:**
```bash
logpipe -file app.json -format logfmt
```

## Text output format

Each line is rendered as:

```
<time> [LEVEL] <message> key=value key=value ...
```

Timestamps are normalised to `HH:MM:SS` (UTC). Well-known field names (`time`, `ts`, `timestamp`, `level`, `lvl`, `severity`, `message`, `msg`, `text`) are extracted into fixed positions; all other fields appear as sorted `key=value` pairs at the end.

When `-color` is enabled, log levels are highlighted:

| Level | Color |
|-------|-------|
| `error` / `err` / `fatal` / `crit` | Bold red |
| `warn` / `warning` | Bold yellow |
| `info` / `information` | Bold green |
| other | Gray |

## Project structure

```
logpipe/
├── cmd/logpipe/       # main package — CLI entry point
├── internal/
│   ├── parser/        # log format parsers (JSON, logfmt)
│   ├── filter/        # field-based entry filtering
│   └── formatter/     # output formatters (text, JSON, logfmt)
└── go.mod
```

## License

MIT
