package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"rtcstats"
)

func main() {
	// Define flags
	outputFile := flag.String("o", "", "Output file (default: stdout)")
	output := flag.String("output", "", "Output file (default: stdout)")
	tsMode := flag.String("ts", "absolute", "Timestamp mode: absolute|delta|both")
	pretty := flag.Bool("pretty", false, "Pretty-print JSON output")
	quiet := flag.Bool("q", false, "Suppress stats output")
	quietLong := flag.Bool("quiet", false, "Suppress stats output")
	sample := flag.Bool("sample", false, "Enable adaptive sampling for getstats events")
	sampleN := flag.Int("sample-n", 5, "Sampling interval: keep every Nth getstats sample")
	sampleCtx := flag.Int("sample-ctx", 2, "Context window: samples before/after interesting moments")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: rtcstats [flags] <input-file>\n\n")
		fmt.Fprintf(os.Stderr, "RTC Stats Pre-Processor - Compresses WebRTC event logs for LLM analysis\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  rtcstats events.jsonl                    Process file, output to stdout\n")
		fmt.Fprintf(os.Stderr, "  rtcstats -o out.jsonl events.jsonl       Process file, output to file\n")
		fmt.Fprintf(os.Stderr, "  rtcstats --ts delta events.jsonl         Use delta timestamps\n")
		fmt.Fprintf(os.Stderr, "  rtcstats --pretty events.jsonl           Pretty-print output\n")
		fmt.Fprintf(os.Stderr, "  rtcstats -q events.jsonl                 Suppress stats logging\n")
		fmt.Fprintf(os.Stderr, "  rtcstats --sample events.jsonl           Enable adaptive sampling\n")
		fmt.Fprintf(os.Stderr, "  rtcstats --sample --sample-n 10 e.jsonl  Sample every 10th getstats\n")
	}

	flag.Parse()

	// Get input file
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: input file required\n\n")
		flag.Usage()
		os.Exit(1)
	}
	inputFile := args[0]

	// Handle output flag aliases
	outPath := *outputFile
	if outPath == "" {
		outPath = *output
	}

	// Parse timestamp mode
	var timestampMode rtcstats.TimestampMode
	switch strings.ToLower(*tsMode) {
	case "absolute", "abs":
		timestampMode = rtcstats.TSAbsolute
	case "delta", "dt":
		timestampMode = rtcstats.TSDelta
	case "both":
		timestampMode = rtcstats.TSBoth
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid timestamp mode: %s (use: absolute|delta|both)\n", *tsMode)
		os.Exit(1)
	}

	// Build options
	opts := []rtcstats.Option{
		rtcstats.WithTimestampMode(timestampMode),
	}
	if *pretty {
		opts = append(opts, rtcstats.WithPrettyPrint())
	}
	if !*quiet && !*quietLong {
		opts = append(opts, rtcstats.WithLogger(rtcstats.StderrLogger()))
	}
	if *sample {
		opts = append(opts, rtcstats.WithSampling())
		if *sampleN != 5 {
			opts = append(opts, rtcstats.WithSamplingInterval(*sampleN))
		}
		if *sampleCtx != 2 {
			opts = append(opts, rtcstats.WithSamplingContext(*sampleCtx, *sampleCtx))
		}
	}

	// Process
	if _, err := rtcstats.ProcessStats(inputFile, outPath, opts...); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
