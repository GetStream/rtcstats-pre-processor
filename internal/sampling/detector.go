package sampling

import "math"

// InterestDetector evaluates whether a compressed getstats output represents
// an "interesting" moment that warrants full-resolution sampling.
type InterestDetector struct {
	prevCategories map[string]map[string]bool // scope → set of category keys last seen
	prevGauges     map[string]map[string]float64 // scope → gauge field → value
}

// NewInterestDetector creates a new detector.
func NewInterestDetector() *InterestDetector {
	return &InterestDetector{
		prevCategories: make(map[string]map[string]bool),
		prevGauges:     make(map[string]map[string]float64),
	}
}

// IsInteresting inspects the compressed getstats output and returns true
// if any trigger condition is met. It operates on the compressed output maps
// to avoid duplicating field extraction logic.
func (d *InterestDetector) IsInteresting(scope string, payload interface{}) bool {
	result, ok := payload.(map[string]interface{})
	if !ok {
		return false
	}

	interesting := false

	// Check for category appearance/disappearance (track added/removed)
	currentKeys := make(map[string]bool, len(result))
	for k := range result {
		currentKeys[k] = true
	}
	if prev, exists := d.prevCategories[scope]; exists {
		if !sameKeys(prev, currentKeys) {
			interesting = true
		}
	}

	// Check trigger fields within each report category
	for catKey, catVal := range result {
		interesting = interesting || d.checkCategory(scope, catKey, catVal)
	}

	// Update previous state
	d.prevCategories[scope] = currentKeys

	return interesting
}

// checkCategory inspects a single report category for trigger conditions.
func (d *InterestDetector) checkCategory(scope, catKey string, catVal interface{}) bool {
	if d.prevGauges[scope] == nil {
		d.prevGauges[scope] = make(map[string]float64)
	}
	gauges := d.prevGauges[scope]

	switch entries := catVal.(type) {
	case map[string]interface{}:
		return d.checkFields(scope, catKey, "", entries, gauges)
	case []interface{}:
		found := false
		for i, item := range entries {
			if m, ok := item.(map[string]interface{}); ok {
				found = found || d.checkFields(scope, catKey, string(rune('0'+i)), m, gauges)
			}
		}
		return found
	case []map[string]interface{}:
		found := false
		for i, m := range entries {
			found = found || d.checkFields(scope, catKey, string(rune('0'+i)), m, gauges)
		}
		return found
	}
	return false
}

// checkFields checks individual compressed fields for trigger conditions.
func (d *InterestDetector) checkFields(scope, catKey, suffix string, fields map[string]interface{}, gauges map[string]float64) bool {
	interesting := false
	prefix := catKey + suffix + "."

	// Counter deltas > 0 triggers (any non-zero delta for these is interesting)
	for _, key := range []string{"pl", "fzc", "fdr"} {
		if v, ok := fields[key]; ok {
			if fv, ok := toFloat(v); ok && fv > 0 {
				interesting = true
			}
		}
	}

	// Gauge change triggers
	gaugeChecks := []struct {
		key       string
		threshold float64
	}{
		{"fps", 5},
		{"j", 0.02},
		{"rtt", 0.05},
		{"s", 10},
	}

	for _, gc := range gaugeChecks {
		if v, ok := fields[gc.key]; ok {
			if fv, ok := toFloat(v); ok {
				gk := prefix + gc.key
				if prev, exists := gauges[gk]; exists {
					if math.Abs(fv-prev) > gc.threshold {
						interesting = true
					}
				}
				gauges[gk] = fv
			}
		}
	}

	// Also check "rtt" within cp (candidate pair) entries — the compressed
	// field name is "rtt" which is already covered above.

	return interesting
}

func sameKeys(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
