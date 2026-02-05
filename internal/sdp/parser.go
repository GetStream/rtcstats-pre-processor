package sdp

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// ParseSDP parses an SDP string and returns structured data
func ParseSDP(sdp string) *ParsedSDP {
	if sdp == "" {
		return nil
	}

	p := &ParsedSDP{
		Raw: sdp,
	}

	lines := strings.Split(sdp, "\r\n")
	if len(lines) <= 1 {
		lines = strings.Split(sdp, "\n")
	}

	var currentMedia *MediaSection

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse attribute lines
		if strings.HasPrefix(line, "a=") {
			attr := line[2:]
			p.parseAttribute(attr, currentMedia)
			continue
		}

		// Parse media lines
		if strings.HasPrefix(line, "m=") {
			media := parseMediaLine(line)
			p.Media = append(p.Media, media)
			currentMedia = &p.Media[len(p.Media)-1]
			continue
		}
	}

	return p
}

// ParsedSDP represents a parsed SDP
type ParsedSDP struct {
	Raw        string
	BundleMIDs []string
	ICELite    bool
	Media      []MediaSection
}

// MediaSection represents an m= section
type MediaSection struct {
	Kind      string   // audio, video, application
	Port      int      // 0 means rejected
	Protocol  string
	Formats   []string // payload types
	MID       string
	Direction string // sendrecv, sendonly, recvonly, inactive
	Codecs    []Codec
	RIDs      []string // simulcast RIDs
	HasTCC    bool     // transport-wide congestion control
}

// Codec represents a codec from rtpmap
type Codec struct {
	PayloadType string
	Name        string
	ClockRate   string
}

var rtpmapPattern = regexp.MustCompile(`^rtpmap:(\d+)\s+([^/]+)/(\d+)`)
var midPattern = regexp.MustCompile(`^mid:(.+)$`)
var ridPattern = regexp.MustCompile(`^rid:(\S+)\s+(send|recv)`)

func (p *ParsedSDP) parseAttribute(attr string, media *MediaSection) {
	// Session-level attributes
	if media == nil {
		if strings.HasPrefix(attr, "group:BUNDLE ") {
			mids := strings.Fields(attr[13:])
			p.BundleMIDs = mids
			return
		}
		if attr == "ice-lite" {
			p.ICELite = true
			return
		}
		return
	}

	// Media-level attributes
	switch {
	case strings.HasPrefix(attr, "mid:"):
		if m := midPattern.FindStringSubmatch(attr); len(m) == 2 {
			media.MID = m[1]
		}

	case strings.HasPrefix(attr, "rtpmap:"):
		if m := rtpmapPattern.FindStringSubmatch(attr); len(m) == 4 {
			codec := Codec{
				PayloadType: m[1],
				Name:        m[2],
				ClockRate:   m[3],
			}
			// Filter out RTX and other repair codecs
			nameLower := strings.ToLower(codec.Name)
			if nameLower != "rtx" && nameLower != "red" && nameLower != "ulpfec" {
				media.Codecs = append(media.Codecs, codec)
			}
		}

	case attr == "sendrecv" || attr == "sendonly" || attr == "recvonly" || attr == "inactive":
		media.Direction = attr

	case strings.HasPrefix(attr, "rid:"):
		if m := ridPattern.FindStringSubmatch(attr); len(m) == 3 {
			media.RIDs = append(media.RIDs, m[1])
		}

	case strings.Contains(attr, "transport-cc"):
		media.HasTCC = true
	}
}

func parseMediaLine(line string) MediaSection {
	// m=video 9 UDP/TLS/RTP/SAVPF 120 124 121
	parts := strings.Fields(line[2:])
	media := MediaSection{
		Direction: "sendrecv", // default
	}

	if len(parts) >= 1 {
		media.Kind = parts[0]
	}
	if len(parts) >= 2 {
		if parts[1] == "0" {
			media.Port = 0 // rejected
		} else {
			media.Port = 9 // typically 9 for bundled
		}
	}
	if len(parts) >= 3 {
		media.Protocol = parts[2]
	}
	if len(parts) >= 4 {
		media.Formats = parts[3:]
	}

	return media
}

// Hash returns a short hash of the SDP for correlation
func (p *ParsedSDP) Hash() string {
	h := sha256.Sum256([]byte(p.Raw))
	return hex.EncodeToString(h[:8]) // 16 hex chars
}

// CodecNames returns just the codec names for a media section (limited to 8)
func (m *MediaSection) CodecNames() []string {
	seen := make(map[string]bool)
	var names []string
	for _, c := range m.Codecs {
		name := strings.ToUpper(c.Name)
		// Normalize common codec names
		switch strings.ToLower(c.Name) {
		case "opus":
			name = "opus"
		case "vp8":
			name = "VP8"
		case "vp9":
			name = "VP9"
		case "h264":
			name = "H264"
		case "av1":
			name = "AV1"
		}
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
			if len(names) >= 8 {
				break
			}
		}
	}
	return names
}
