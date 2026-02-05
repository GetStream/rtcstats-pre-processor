package handlers

import (
	"encoding/json"
	"strings"

	"rtcstats/internal/event"
	"rtcstats/internal/sdp"
	"rtcstats/internal/transform"
)

// CreatePCHandler handles the "create" event for PeerConnection creation
type CreatePCHandler struct{}

func (h *CreatePCHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	// Bundle policy
	if bp, ok := payload["bundlePolicy"].(string); ok {
		// Shorten common values
		switch bp {
		case "max-bundle":
			result["bp"] = "mb"
		case "max-compat":
			result["bp"] = "mc"
		case "balanced":
			result["bp"] = "b"
		default:
			result["bp"] = bp
		}
	}

	// ICE servers summary
	if servers, ok := payload["iceServers"].([]interface{}); ok {
		iceSummary := summarizeICEServers(servers)
		if len(iceSummary) > 0 {
			result["ice"] = iceSummary
		}
	}

	return result
}

func summarizeICEServers(servers []interface{}) map[string]int {
	summary := map[string]int{}
	hosts := make(map[string]bool)

	for _, s := range servers {
		server, ok := s.(map[string]interface{})
		if !ok {
			continue
		}

		urls, ok := server["urls"].([]interface{})
		if !ok {
			// Single URL as string
			if url, ok := server["urls"].(string); ok {
				urls = []interface{}{url}
			} else {
				continue
			}
		}

		for _, u := range urls {
			url, ok := u.(string)
			if !ok {
				continue
			}

			// Parse URL scheme and transport
			if strings.HasPrefix(url, "turns:") {
				summary["turns"]++
			} else if strings.HasPrefix(url, "turn:") {
				summary["turn"]++
			} else if strings.HasPrefix(url, "stun:") {
				summary["stun"]++
			}

			// Check transport
			if strings.Contains(url, "transport=tcp") {
				summary["tcp"]++
			} else if strings.Contains(url, "transport=udp") || !strings.Contains(url, "transport=") {
				summary["udp"]++
			}

			// Extract host for deduplication
			host := extractHost(url)
			if host != "" {
				hosts[host] = true
			}
		}
	}

	if len(hosts) > 0 {
		summary["hosts"] = len(hosts)
	}

	return summary
}

func extractHost(url string) string {
	// Remove scheme
	url = strings.TrimPrefix(url, "turns:")
	url = strings.TrimPrefix(url, "turn:")
	url = strings.TrimPrefix(url, "stun:")

	// Remove port and params
	if idx := strings.Index(url, ":"); idx > 0 {
		url = url[:idx]
	}
	if idx := strings.Index(url, "?"); idx > 0 {
		url = url[:idx]
	}

	return url
}

// CreateOfferHandler handles createOffer events
type CreateOfferHandler struct{}

func (h *CreateOfferHandler) Transform(e event.RawEvent) interface{} {
	// createOffer typically has [null] or empty options
	return nil
}

// CreateOfferSuccessHandler handles createOfferOnSuccess events
type CreateOfferSuccessHandler struct{}

func (h *CreateOfferSuccessHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	sdpType := "offer"
	if t, ok := payload["type"].(string); ok {
		sdpType = t
	}
	result["t"] = "o"

	if sdpStr, ok := payload["sdp"].(string); ok {
		digest := sdp.CreateSDPDigest(sdpStr, sdpType)
		if digest != nil {
			result["sdp_sum"] = digest
		}
	}

	return result
}

// CreateAnswerHandler handles createAnswer events
type CreateAnswerHandler struct{}

func (h *CreateAnswerHandler) Transform(e event.RawEvent) interface{} {
	return nil
}

// CreateAnswerSuccessHandler handles createAnswerOnSuccess events
type CreateAnswerSuccessHandler struct{}

func (h *CreateAnswerSuccessHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	sdpType := "answer"
	if t, ok := payload["type"].(string); ok {
		sdpType = t
	}
	result["t"] = "a"

	if sdpStr, ok := payload["sdp"].(string); ok {
		digest := sdp.CreateSDPDigest(sdpStr, sdpType)
		if digest != nil {
			result["sdp_sum"] = digest
		}
	}

	return result
}

// SetDescriptionHandler handles setLocalDescription and setRemoteDescription
type SetDescriptionHandler struct {
	IsLocal bool
}

func (h *SetDescriptionHandler) Transform(e event.RawEvent) interface{} {
	// Payload is typically an array with one element
	var payloadArr []map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payloadArr); err != nil {
		// Try as single object
		var payload map[string]interface{}
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			return nil
		}
		payloadArr = []map[string]interface{}{payload}
	}

	if len(payloadArr) == 0 {
		return nil
	}

	payload := payloadArr[0]
	result := make(map[string]interface{})

	sdpType := "offer"
	if t, ok := payload["type"].(string); ok {
		sdpType = t
		if sdpType == "offer" {
			result["t"] = "o"
		} else {
			result["t"] = "a"
		}
	}

	if sdpStr, ok := payload["sdp"].(string); ok {
		digest := sdp.CreateSDPDigest(sdpStr, sdpType)
		if digest != nil {
			result["sdp_sum"] = digest
		}
	}

	return result
}

// OnTrackHandler handles ontrack events
type OnTrackHandler struct{}

func (h *OnTrackHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	if kind, ok := payload["kind"].(string); ok {
		result["k"] = transform.CompressMediaKind(kind)
	}

	// Try to get mid from track or transceiver
	if mid, ok := payload["mid"].(string); ok {
		result["mid"] = mid
	}

	return result
}

