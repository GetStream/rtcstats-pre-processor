package sampling

import "reflect"

// SteadyStateSuppressor replaces unchanged report categories with "="
// to further reduce output size for long, stable calls.
type SteadyStateSuppressor struct {
	lastEmitted map[string]interface{} // scope → category key → last emitted value
}

// NewSteadyStateSuppressor creates a new suppressor.
func NewSteadyStateSuppressor() *SteadyStateSuppressor {
	return &SteadyStateSuppressor{
		lastEmitted: make(map[string]interface{}),
	}
}

// Suppress compares each category in the result against the last emitted
// version for the given scope. If a category is identical, it is replaced
// with the string "=". Returns the (possibly modified) result.
func (s *SteadyStateSuppressor) Suppress(scope string, payload interface{}) interface{} {
	result, ok := payload.(map[string]interface{})
	if !ok {
		return payload
	}

	key := scope
	prev, hasPrev := s.lastEmitted[key]
	var prevMap map[string]interface{}
	if hasPrev {
		prevMap, _ = prev.(map[string]interface{})
	}

	// Build a copy of the result with suppression applied
	suppressed := make(map[string]interface{}, len(result))
	// Also build the full version for next comparison
	full := make(map[string]interface{}, len(result))

	for catKey, catVal := range result {
		full[catKey] = catVal
		if prevMap != nil {
			if prevCat, exists := prevMap[catKey]; exists {
				if reflect.DeepEqual(catVal, prevCat) {
					suppressed[catKey] = "="
					continue
				}
			}
		}
		suppressed[catKey] = catVal
	}

	// Store the full (unsuppressed) version for future comparison
	s.lastEmitted[key] = full

	return suppressed
}
