# rtcstats-pre-processor

A Go library and CLI tool that compresses WebRTC event logs into a compact format optimized for LLM token consumption.

Given raw rtcstats JSONL dumps (arrays of `[eventName, scope, payload, timestamp]`), the processor:

- Abbreviates field names (`n`, `s`, `p`, `ts`, `dt`)
- Extracts and flattens handler-specific payloads (SDP digests, ICE candidates, stats reports)
- Supports absolute, delta, or both timestamp modes
- Reports input/output size with reduction percentage

## Installation

**CLI:**

```bash
go install rtcstats/cmd/rtcstats@latest
```

**Library:**

```bash
go get rtcstats
```

## CLI Usage

```
rtcstats [flags] <input-file>
```

| Flag | Description |
|------|-------------|
| `-o`, `--output` | Output file (default: stdout) |
| `--ts` | Timestamp mode: `absolute`\|`delta`\|`both` (default: `absolute`) |
| `--pretty` | Pretty-print JSON output |
| `-q`, `--quiet` | Suppress stats logging to stderr |

**Examples:**

```bash
# Process file, output to stdout
rtcstats events.jsonl

# Write to file
rtcstats -o compressed.jsonl events.jsonl

# Delta timestamps, pretty-printed
rtcstats --ts delta --pretty events.jsonl

# Pipe to another tool, suppress stats
rtcstats -q events.jsonl | jq .

# Discard output, show stats only
rtcstats -o /dev/null events.jsonl
```

## Package Usage

### File-to-file

```go
import "rtcstats"

result, err := rtcstats.ProcessStats("input.jsonl", "output.jsonl",
    rtcstats.WithTimestampMode(rtcstats.TSDelta),
    rtcstats.WithLogger(rtcstats.StderrLogger()),
)
// result.Reduction => 0.73 (73% smaller)
```

### Streaming (io.Reader / io.Writer)

Works with HTTP handlers, stdin/stdout piping, or any io stream.

```go
import "rtcstats"

result, err := rtcstats.Process(r, w,
    rtcstats.WithTimestampMode(rtcstats.TSAbsolute),
)
```

### In-memory

Useful for serverless functions, tests, or batch processing.

```go
import "rtcstats"

output, result, err := rtcstats.ProcessBytes(inputBytes,
    rtcstats.WithPrettyPrint(),
)
```

### Stats-only analysis

```go
import (
    "io"
    "rtcstats"
)

result, err := rtcstats.Process(file, io.Discard)
fmt.Printf("%d events, %.0f%% reduction\n", result.EventCount, result.Reduction*100)
```

## Options

| Function | Description |
|----------|-------------|
| `WithTimestampMode(mode)` | `TSAbsolute` (default), `TSDelta`, or `TSBoth` |
| `WithPrettyPrint()` | Indent JSON output |
| `WithLogger(l)` | Receive stats log line after processing |

## LLM Prompt Injection

The `internal/prompts` package exports constant strings that translate compressed field names back to human-readable descriptions. Inject these into your LLM system prompts so the model can interpret abbreviated output.

```go
import "rtcstats/internal/prompts"

// Full reference covering stats, events, SDP digests, and scopes
systemPrompt := "You are a WebRTC diagnostics assistant.\n\n" + prompts.FullReference

// Or pick only what you need:
//   prompts.StatsFields      – getstats report field translations (out_v, in_a, etc.)
//   prompts.EventFields      – connection event payload field translations
//   prompts.SDPDigestFields   – SDP summary (sdp_sum) field translations
//   prompts.ScopeReference   – scope string conventions (0-pub, 0-sub, sfu:*)
```

Available constants:

| Constant | Covers |
|----------|--------|
| `prompts.StatsFields` | All getstats report types and their abbreviated fields (bs, hbs, fps, etc.) |
| `prompts.EventFields` | Connection event payload keys (did, sid, uid, ok, dur, etc.) and state enums |
| `prompts.SDPDigestFields` | SDP digest object fields (sdp_sum: type, codecs, sim_rids, tcc, etc.) |
| `prompts.ScopeReference` | Scope string meanings (0-pub, 0-sub, sfu:\<region\>) |
| `prompts.FullReference` | All of the above concatenated |

## Result

`ProcessStats`, `Process`, and `ProcessBytes` all return a `*Result`:

```go
type Result struct {
    InputBytes  int64   // raw input size
    OutputBytes int64   // compressed output size
    Reduction   float64 // 0-1 fraction (e.g. 0.73 = 73% reduction)
    EventCount  int     // number of events processed
}
```
