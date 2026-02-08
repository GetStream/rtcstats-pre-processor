package sampling

import (
	"rtcstats/internal/event"
	"rtcstats/internal/handlers"
)

// EmitFunc is called when the sampler decides to emit a getstats event.
// It receives the compressed event and snapshot for delta recomputation.
type EmitFunc func(ce event.CompressedEvent, snapshot *handlers.StatsSnapshot)

// bufferedSample holds a compressed event in the ring buffer.
type bufferedSample struct {
	event    event.CompressedEvent
	snapshot *handlers.StatsSnapshot
	keep     bool // true = force-emit even if not Nth
}

// scopeState tracks per-scope sampling state.
type scopeState struct {
	count        int              // total samples seen
	contextAfter int              // countdown of remaining context-after samples
	buffer       []bufferedSample // ring buffer (max size = contextBefore + 1)
}

// Sampler implements two-layer adaptive sampling for getstats events.
type Sampler struct {
	config   Config
	detector *InterestDetector
	scopes   map[string]*scopeState
	emitFunc EmitFunc
}

// NewSampler creates a sampler with the given config and emission callback.
func NewSampler(config Config, emitFunc EmitFunc) *Sampler {
	return &Sampler{
		config:   config,
		detector: NewInterestDetector(),
		scopes:   make(map[string]*scopeState),
		emitFunc: emitFunc,
	}
}

// ProcessGetStats evaluates a compressed getstats event and either buffers,
// emits, or discards it based on the adaptive sampling algorithm.
func (s *Sampler) ProcessGetStats(ce event.CompressedEvent, payload interface{}, snapshot *handlers.StatsSnapshot) {
	scope := ce.Scope
	st := s.scopes[scope]
	if st == nil {
		st = &scopeState{}
		s.scopes[scope] = st
	}
	st.count++

	interesting := s.detector.IsInteresting(scope, payload)

	// Determine whether to keep this sample
	keep := false
	if st.count == 1 {
		// Always keep the first sample
		keep = true
	} else if st.count%s.config.Interval == 0 {
		// Keep every Nth sample
		keep = true
	} else if interesting {
		keep = true
	} else if st.contextAfter > 0 {
		// Within context-after window of a previous interesting moment
		keep = true
		st.contextAfter--
	}

	if interesting {
		// Retroactively mark all buffered samples as keep
		for i := range st.buffer {
			st.buffer[i].keep = true
		}
		// Start context-after countdown
		st.contextAfter = s.config.ContextAfter
	}

	sample := bufferedSample{
		event:    ce,
		snapshot: snapshot,
		keep:     keep,
	}

	bufSize := s.config.ContextBefore + 1
	st.buffer = append(st.buffer, sample)

	// Pop and process the oldest when buffer exceeds capacity
	for len(st.buffer) > bufSize {
		oldest := st.buffer[0]
		st.buffer = st.buffer[1:]
		if oldest.keep {
			s.emitFunc(oldest.event, oldest.snapshot)
		}
	}
}

// Flush drains all buffers, force-keeping the last sample per scope.
func (s *Sampler) Flush() {
	for _, st := range s.scopes {
		if len(st.buffer) == 0 {
			continue
		}
		// Force-keep the last sample
		st.buffer[len(st.buffer)-1].keep = true

		for _, sample := range st.buffer {
			if sample.keep {
				s.emitFunc(sample.event, sample.snapshot)
			}
		}
		st.buffer = nil
	}
}
