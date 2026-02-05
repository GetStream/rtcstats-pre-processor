package processor

import (
	"io"

	"rtcstats/internal/event"
	"rtcstats/internal/handlers"
	"rtcstats/internal/transform"
)

// Pipeline processes RawEvents and outputs CompressedEvents
type Pipeline struct {
	reader   *event.Reader
	writer   *event.Writer
	registry *handlers.Registry
	tsMode   event.TimestampMode
	firstTS  int64
}

// NewPipeline creates a new processing pipeline
func NewPipeline(reader *event.Reader, w io.Writer, tsMode event.TimestampMode, pretty bool) *Pipeline {
	return &Pipeline{
		reader:   reader,
		writer:   event.NewWriter(w, pretty),
		registry: handlers.NewRegistry(),
		tsMode:   tsMode,
	}
}

// Run processes all events
func (p *Pipeline) Run() error {
	events := p.reader.AllEvents()

	for i, rawEvent := range events {
		// Track first timestamp for delta calculation
		if i == 0 {
			p.firstTS = rawEvent.TS
		}

		compressed := p.transformEvent(rawEvent)
		if err := p.writer.Write(compressed); err != nil {
			return err
		}
	}

	return nil
}

func (p *Pipeline) transformEvent(raw event.RawEvent) event.CompressedEvent {
	// Get the appropriate handler
	handler := p.registry.Get(raw.Name)
	payload := handler.Transform(raw)

	// Build compressed event
	compressed := event.CompressedEvent{
		Name:    raw.Name,
		Scope:   transform.CompressScope(raw.Scope),
		Payload: payload,
	}

	// Handle timestamps based on mode
	switch p.tsMode {
	case event.TSAbsolute:
		compressed.TS = raw.TS
	case event.TSDelta:
		dt := raw.TS - p.firstTS
		compressed.DT = &dt
	case event.TSBoth:
		compressed.TS = raw.TS
		dt := raw.TS - p.firstTS
		compressed.DT = &dt
	}

	return compressed
}
