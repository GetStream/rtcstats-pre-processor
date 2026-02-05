package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Reader reads RawEvents from JSON or JSONL files
type Reader struct {
	events []RawEvent
}

// NewReaderFromFile creates a Reader from a file path
func NewReaderFromFile(path string) (*Reader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return NewReader(data)
}

// NewReader creates a Reader from raw bytes, auto-detecting JSON vs JSONL
func NewReader(data []byte) (*Reader, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return &Reader{events: []RawEvent{}}, nil
	}

	// Use streaming JSON decoder - handles both single-line and multi-line JSONL
	if data[0] == '[' {
		return parseJSONStream(data)
	}

	return nil, fmt.Errorf("expected JSON array, got: %c", data[0])
}

func parseJSONStream(data []byte) (*Reader, error) {
	var events []RawEvent

	// Use a streaming approach to handle multi-line JSON arrays
	decoder := json.NewDecoder(bytes.NewReader(data))

	eventNum := 0
	for decoder.More() {
		eventNum++
		var raw []json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, fmt.Errorf("event %d: %w", eventNum, err)
		}

		if len(raw) < 4 {
			return nil, fmt.Errorf("event %d: array has %d elements, need 4", eventNum, len(raw))
		}

		var event RawEvent

		// Parse event name (string)
		if err := json.Unmarshal(raw[0], &event.Name); err != nil {
			return nil, fmt.Errorf("event %d: parsing event name: %w", eventNum, err)
		}

		// Parse scope (nullable string)
		if string(raw[1]) != "null" {
			var scope string
			if err := json.Unmarshal(raw[1], &scope); err != nil {
				return nil, fmt.Errorf("event %d: parsing scope: %w", eventNum, err)
			}
			event.Scope = &scope
		}

		// Keep payload as raw JSON
		event.Payload = raw[2]

		// Parse timestamp (int64)
		if err := json.Unmarshal(raw[3], &event.TS); err != nil {
			return nil, fmt.Errorf("event %d: parsing timestamp: %w", eventNum, err)
		}

		events = append(events, event)
	}

	return &Reader{events: events}, nil
}

// Events returns a channel that yields events
func (r *Reader) Events() <-chan RawEvent {
	ch := make(chan RawEvent)
	go func() {
		defer close(ch)
		for _, e := range r.events {
			ch <- e
		}
	}()
	return ch
}

// AllEvents returns all events as a slice
func (r *Reader) AllEvents() []RawEvent {
	return r.events
}

// Writer writes CompressedEvents as JSONL
type Writer struct {
	w      io.Writer
	pretty bool
}

// NewWriter creates a new Writer
func NewWriter(w io.Writer, pretty bool) *Writer {
	return &Writer{w: w, pretty: pretty}
}

// Write outputs a CompressedEvent as JSON
func (w *Writer) Write(e CompressedEvent) error {
	var data []byte
	var err error

	if w.pretty {
		data, err = json.MarshalIndent(e, "", "  ")
	} else {
		data, err = json.Marshal(e)
	}

	if err != nil {
		return err
	}

	_, err = w.w.Write(append(data, '\n'))
	return err
}
