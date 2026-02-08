package sampling

// Config controls adaptive sampling behavior for getstats events.
type Config struct {
	Enabled       bool // enable adaptive sampling
	Interval      int  // keep every Nth getstats sample (default 5)
	ContextBefore int  // full-resolution samples before interesting moment (default 2)
	ContextAfter  int  // full-resolution samples after interesting moment (default 2)
	SteadyState   bool // replace unchanged report categories with "=" (default true)
}

// DefaultConfig returns a Config with recommended defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:       true,
		Interval:      5,
		ContextBefore: 2,
		ContextAfter:  2,
		SteadyState:   true,
	}
}
