package rtcstats

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"rtcstats/internal/event"
	"rtcstats/internal/ioutil"
	"rtcstats/internal/processor"
	"rtcstats/internal/sampling"
)

// TimestampMode controls how timestamps appear in output.
type TimestampMode = event.TimestampMode

const (
	TSAbsolute TimestampMode = event.TSAbsolute
	TSDelta    TimestampMode = event.TSDelta
	TSBoth     TimestampMode = event.TSBoth
)

// Result holds processing statistics.
type Result struct {
	InputBytes  int64
	OutputBytes int64
	Reduction   float64 // 0–1 fraction
	EventCount  int
}

// Logger receives processing stats. Compatible with log.Printf.
type Logger interface {
	Printf(format string, args ...interface{})
}

// stderrLogger writes to os.Stderr.
type stderrLogger struct{}

func (stderrLogger) Printf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// StderrLogger returns a Logger that writes to stderr.
func StderrLogger() Logger { return stderrLogger{} }

// Option configures processing behavior.
type Option func(*options)

type options struct {
	tsMode   TimestampMode
	pretty   bool
	logger   Logger
	sampling *sampling.Config
}

// WithTimestampMode sets absolute, delta, or both.
func WithTimestampMode(mode TimestampMode) Option {
	return func(o *options) { o.tsMode = mode }
}

// WithPrettyPrint enables indented JSON output.
func WithPrettyPrint() Option {
	return func(o *options) { o.pretty = true }
}

// WithLogger sets a Logger to receive file-size stats after processing.
func WithLogger(l Logger) Option {
	return func(o *options) { o.logger = l }
}

// WithSampling enables adaptive sampling with default settings
// (interval=5, context=2, steady-state=true).
func WithSampling() Option {
	return func(o *options) {
		cfg := sampling.DefaultConfig()
		o.sampling = &cfg
	}
}

// WithSamplingInterval sets the sampling interval (keep every Nth getstats).
// Implies WithSampling().
func WithSamplingInterval(n int) Option {
	return func(o *options) {
		if o.sampling == nil {
			cfg := sampling.DefaultConfig()
			o.sampling = &cfg
		}
		o.sampling.Interval = n
	}
}

// WithSamplingContext sets the context window (samples before/after interesting moments).
// Implies WithSampling().
func WithSamplingContext(before, after int) Option {
	return func(o *options) {
		if o.sampling == nil {
			cfg := sampling.DefaultConfig()
			o.sampling = &cfg
		}
		o.sampling.ContextBefore = before
		o.sampling.ContextAfter = after
	}
}

func applyOpts(opts []Option) options {
	o := options{tsMode: TSAbsolute}
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// ProcessStats reads inputPath, processes events, and writes to outputPath.
// If outputPath is "" or "-", it writes to stdout.
func ProcessStats(inputPath, outputPath string, opts ...Option) (*Result, error) {
	cfg := applyOpts(opts)

	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	inputSize := int64(len(inputData))

	reader, err := event.NewReader(inputData)
	if err != nil {
		return nil, fmt.Errorf("parsing input: %w", err)
	}

	var dest io.Writer = os.Stdout
	var outFile *os.File
	if outputPath != "" && outputPath != "-" {
		outFile, err = os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("creating output file: %w", err)
		}
		defer outFile.Close()
		dest = outFile
	}

	cw := &ioutil.CountWriter{W: dest}
	pipeline := processor.NewPipeline(reader, cw, cfg.tsMode, cfg.pretty, cfg.sampling)
	if err := pipeline.Run(); err != nil {
		return nil, fmt.Errorf("processing: %w", err)
	}

	res := buildResult(inputSize, cw.Count, len(reader.AllEvents()))
	logResult(cfg.logger, res, inputPath, outputPath)
	return res, nil
}

// Process reads from r, processes events, and writes to w.
func Process(r io.Reader, w io.Writer, opts ...Option) (*Result, error) {
	cfg := applyOpts(opts)

	inputData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	inputSize := int64(len(inputData))

	reader, err := event.NewReader(inputData)
	if err != nil {
		return nil, fmt.Errorf("parsing input: %w", err)
	}

	cw := &ioutil.CountWriter{W: w}
	pipeline := processor.NewPipeline(reader, cw, cfg.tsMode, cfg.pretty, cfg.sampling)
	if err := pipeline.Run(); err != nil {
		return nil, fmt.Errorf("processing: %w", err)
	}

	res := buildResult(inputSize, cw.Count, len(reader.AllEvents()))
	logResult(cfg.logger, res, "", "")
	return res, nil
}

// ProcessBytes processes input bytes in memory and returns the output bytes.
func ProcessBytes(input []byte, opts ...Option) ([]byte, *Result, error) {
	cfg := applyOpts(opts)

	inputSize := int64(len(input))
	reader, err := event.NewReader(input)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing input: %w", err)
	}

	var buf bytes.Buffer
	cw := &ioutil.CountWriter{W: &buf}
	pipeline := processor.NewPipeline(reader, cw, cfg.tsMode, cfg.pretty, cfg.sampling)
	if err := pipeline.Run(); err != nil {
		return nil, nil, fmt.Errorf("processing: %w", err)
	}

	res := buildResult(inputSize, cw.Count, len(reader.AllEvents()))
	logResult(cfg.logger, res, "", "")
	return buf.Bytes(), res, nil
}

func buildResult(inputBytes, outputBytes int64, eventCount int) *Result {
	var reduction float64
	if inputBytes > 0 {
		reduction = 1 - float64(outputBytes)/float64(inputBytes)
	}
	return &Result{
		InputBytes:  inputBytes,
		OutputBytes: outputBytes,
		Reduction:   reduction,
		EventCount:  eventCount,
	}
}

func logResult(l Logger, r *Result, inPath, outPath string) {
	if l == nil {
		return
	}
	src := "input"
	if inPath != "" {
		src = inPath
	}
	dst := "output"
	if outPath != "" && outPath != "-" {
		dst = outPath
	}
	l.Printf("%s: %s -> %s: %s (%.1f%% reduction, %d events)",
		src, humanBytes(r.InputBytes),
		dst, humanBytes(r.OutputBytes),
		r.Reduction*100, r.EventCount)
}

func humanBytes(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
