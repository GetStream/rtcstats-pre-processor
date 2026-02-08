package processor

import (
	"io"

	"rtcstats/internal/event"
	"rtcstats/internal/handlers"
	"rtcstats/internal/sampling"
	"rtcstats/internal/transform"
)

// Pipeline processes RawEvents and outputs CompressedEvents
type Pipeline struct {
	reader      *event.Reader
	writer      *event.Writer
	registry    *handlers.Registry
	tsMode      event.TimestampMode
	firstTS     int64
	sampler     *sampling.Sampler
	suppressor  *sampling.SteadyStateSuppressor
	samplingCfg *sampling.Config
	gsHandler   *handlers.GetStatsHandler
	writeErr    error // captures write errors from sampler callback
}

// NewPipeline creates a new processing pipeline
func NewPipeline(reader *event.Reader, w io.Writer, tsMode event.TimestampMode, pretty bool, samplingCfg *sampling.Config) *Pipeline {
	reg := handlers.NewRegistry()
	p := &Pipeline{
		reader:      reader,
		writer:      event.NewWriter(w, pretty),
		registry:    reg,
		tsMode:      tsMode,
		samplingCfg: samplingCfg,
		gsHandler:   reg.GetStatsHandler(),
	}

	if samplingCfg != nil && samplingCfg.Enabled {
		if samplingCfg.SteadyState {
			p.suppressor = sampling.NewSteadyStateSuppressor()
		}
		p.sampler = sampling.NewSampler(*samplingCfg, p.emitSampledEvent)
	}

	return p
}

// Run processes all events
func (p *Pipeline) Run() error {
	events := p.reader.AllEvents()

	for i, rawEvent := range events {
		// Track first timestamp for delta calculation
		if i == 0 {
			p.firstTS = rawEvent.TS
		}

		if p.sampler != nil && rawEvent.Name == "getstats" {
			if err := p.processGetstatsWithSampling(rawEvent); err != nil {
				return err
			}
		} else {
			compressed := p.transformEvent(rawEvent)
			if err := p.writer.Write(compressed); err != nil {
				return err
			}
		}
	}

	// Flush sampler buffers
	if p.sampler != nil {
		p.sampler.Flush()
		if p.writeErr != nil {
			return p.writeErr
		}
	}

	return nil
}

// processGetstatsWithSampling routes a getstats event through the sampler.
func (p *Pipeline) processGetstatsWithSampling(raw event.RawEvent) error {
	// Use ExtractAndTransform to get both payload and snapshot
	payload, snapshot := p.gsHandler.ExtractAndTransform(raw)

	// Build the compressed event envelope (without payload — sampler may recompute it)
	ce := event.CompressedEvent{
		Name:    raw.Name,
		Scope:   transform.CompressScope(raw.Scope),
		Payload: payload,
	}
	switch p.tsMode {
	case event.TSAbsolute:
		ce.TS = raw.TS
	case event.TSDelta:
		dt := raw.TS - p.firstTS
		ce.DT = &dt
	case event.TSBoth:
		ce.TS = raw.TS
		dt := raw.TS - p.firstTS
		ce.DT = &dt
	}

	// Hand to sampler — it will call emitSampledEvent when ready
	p.sampler.ProcessGetStats(ce, payload, snapshot)
	return p.writeErr
}

// emitSampledEvent is the callback from the sampler when it decides to emit.
func (p *Pipeline) emitSampledEvent(ce event.CompressedEvent, snapshot *handlers.StatsSnapshot) {
	if p.writeErr != nil {
		return
	}

	// Recompute deltas against lastEmittedValues baseline
	recomputed := p.gsHandler.RecomputeForEmission(snapshot)

	// Apply steady-state suppression if enabled
	if p.suppressor != nil && recomputed != nil {
		recomputed = p.suppressor.Suppress(ce.Scope, recomputed)
	}

	ce.Payload = recomputed

	// Update the emission baseline
	p.gsHandler.UpdateEmittedBaseline(snapshot)

	if err := p.writer.Write(ce); err != nil {
		p.writeErr = err
	}
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
