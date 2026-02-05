package sdp

// Digest represents the sdp_sum output format
type Digest struct {
	Type       string       `json:"type"`
	SDPHash    string       `json:"sdp_hash,omitempty"`
	BundleMIDs []string     `json:"bundle_mids,omitempty"`
	ICELite    bool         `json:"ice_lite,omitempty"`
	Media      []MediaEntry `json:"media,omitempty"`
}

// MediaEntry represents a single m-line in the digest
type MediaEntry struct {
	MID      string   `json:"mid"`
	Kind     string   `json:"kind"`
	Dir      string   `json:"dir"`
	Rejected bool     `json:"rejected,omitempty"`
	Codecs   []string `json:"codecs,omitempty"`
	SimRIDs  int      `json:"sim_rids,omitempty"`
	TCC      bool     `json:"tcc,omitempty"`
}

// NewDigest creates a Digest from a parsed SDP
func NewDigest(parsed *ParsedSDP, sdpType string) *Digest {
	if parsed == nil {
		return nil
	}

	d := &Digest{
		Type:       sdpType,
		SDPHash:    parsed.Hash(),
		BundleMIDs: parsed.BundleMIDs,
	}

	if parsed.ICELite {
		d.ICELite = true
	}

	for _, m := range parsed.Media {
		entry := MediaEntry{
			MID:      m.MID,
			Kind:     m.Kind,
			Dir:      m.Direction,
			Rejected: m.Port == 0,
			Codecs:   m.CodecNames(),
		}

		if len(m.RIDs) > 0 {
			entry.SimRIDs = len(m.RIDs)
		}

		if m.HasTCC {
			entry.TCC = true
		}

		d.Media = append(d.Media, entry)
	}

	return d
}

// CreateSDPDigest is a convenience function to parse SDP and create a digest
func CreateSDPDigest(sdp string, sdpType string) *Digest {
	parsed := ParseSDP(sdp)
	return NewDigest(parsed, sdpType)
}
