package ice

import (
	"encoding/json"
	"regexp"
	"strings"
)

// CandidateSummary represents a compressed ICE candidate
type CandidateSummary struct {
	Type      string `json:"t,omitempty"`   // host, srflx, relay
	Transport string `json:"tr,omitempty"`  // udp, tcp
	MID       string `json:"mid,omitempty"` // media line index
	EOC       int    `json:"eoc,omitempty"` // end-of-candidates marker
	N         int    `json:"n,omitempty"`   // count (for simple mode)
}

// candidate:0 1 UDP 2122252543 192.168.50.234 51101 typ host
// candidate:4 1 UDP 8331263 89.222.124.8 40846 typ relay raddr 89.222.124.8 rport 40846
var candidatePattern = regexp.MustCompile(`candidate:\S+\s+\d+\s+(\S+)\s+\d+\s+\S+\s+\d+\s+typ\s+(\S+)`)

// ParseCandidate parses an ICE candidate string
func ParseCandidate(candidate string) *CandidateSummary {
	if candidate == "" {
		return &CandidateSummary{EOC: 1}
	}

	matches := candidatePattern.FindStringSubmatch(candidate)
	if len(matches) < 3 {
		return &CandidateSummary{N: 1}
	}

	return &CandidateSummary{
		Type:      matches[2], // host, srflx, prflx, relay
		Transport: strings.ToLower(matches[1]),
	}
}

// ParseCandidateFromPayload extracts and parses candidate from various payload formats
func ParseCandidateFromPayload(payload interface{}) *CandidateSummary {
	switch p := payload.(type) {
	case nil:
		return &CandidateSummary{EOC: 1}

	case string:
		return ParseCandidate(p)

	case map[string]interface{}:
		// Check for candidate field
		if candidate, ok := p["candidate"].(string); ok {
			if candidate == "" {
				return &CandidateSummary{EOC: 1}
			}
			summary := ParseCandidate(candidate)
			if mid, ok := p["sdpMid"].(string); ok {
				summary.MID = mid
			}
			return summary
		}

		// Check for iceCandidate field (nested JSON string)
		if iceCand, ok := p["iceCandidate"].(string); ok {
			var nested map[string]interface{}
			if err := json.Unmarshal([]byte(iceCand), &nested); err == nil {
				return ParseCandidateFromPayload(nested)
			}
		}

		return &CandidateSummary{N: 1}

	default:
		return &CandidateSummary{N: 1}
	}
}

// IsEndOfCandidates checks if this represents end-of-candidates
func (c *CandidateSummary) IsEndOfCandidates() bool {
	return c.EOC == 1
}

// SimpleSummary returns a simple count-based summary
func SimpleSummary() map[string]interface{} {
	return map[string]interface{}{"n": 1}
}

// EOCSummary returns an end-of-candidates summary
func EOCSummary() map[string]interface{} {
	return map[string]interface{}{"eoc": 1}
}

// ToMap converts CandidateSummary to a map for JSON output
func (c *CandidateSummary) ToMap() map[string]interface{} {
	if c.EOC == 1 {
		return EOCSummary()
	}

	if c.Type == "" && c.N > 0 {
		return SimpleSummary()
	}

	result := make(map[string]interface{})
	if c.Type != "" {
		result["t"] = c.Type
	}
	if c.Transport != "" {
		result["tr"] = c.Transport
	}
	if c.MID != "" {
		result["mid"] = c.MID
	}
	return result
}
