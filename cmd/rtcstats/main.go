package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"rtcstats/internal/event"
	"rtcstats/internal/processor"
)

func main() {
	// Define flags
	outputFile := flag.String("o", "", "Output file (default: stdout)")
	output := flag.String("output", "", "Output file (default: stdout)")
	tsMode := flag.String("ts", "absolute", "Timestamp mode: absolute|delta|both")
	pretty := flag.Bool("pretty", false, "Pretty-print JSON output")

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
	var timestampMode event.TimestampMode
	switch strings.ToLower(*tsMode) {
	case "absolute", "abs":
		timestampMode = event.TSAbsolute
	case "delta", "dt":
		timestampMode = event.TSDelta
	case "both":
		timestampMode = event.TSBoth
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid timestamp mode: %s (use: absolute|delta|both)\n", *tsMode)
		os.Exit(1)
	}

	// Read input file
	reader, err := event.NewReaderFromFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	// Set up output writer
	var writer io.Writer = os.Stdout
	if outPath != "" {
		f, err := os.Create(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		writer = f
	}

	// Create and run pipeline
	pipeline := processor.NewPipeline(reader, writer, timestampMode, *pretty)
	if err := pipeline.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing events: %v\n", err)
		os.Exit(1)
	}
}
