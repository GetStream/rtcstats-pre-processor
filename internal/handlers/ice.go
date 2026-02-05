package handlers

import (
	"encoding/json"

	"rtcstats/internal/event"
	"rtcstats/internal/ice"
)

// OnIceCandidateHandler handles onicecandidate events
type OnIceCandidateHandler struct{}

func (h *OnIceCandidateHandler) Transform(e event.RawEvent) interface{} {
	// Check for null payload (end of candidates)
	if len(e.Payload) == 0 || string(e.Payload) == "null" {
		return ice.EOCSummary()
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return ice.SimpleSummary()
	}

	// Check for empty or null candidate (end of candidates)
	if candidate, ok := payload["candidate"].(string); ok {
		if candidate == "" {
			return ice.EOCSummary()
		}
	} else if payload["candidate"] == nil {
		return ice.EOCSummary()
	}

	// Return simple count - candidate details are stripped
	return ice.SimpleSummary()
}

// AddIceCandidateHandler handles addIceCandidate events
type AddIceCandidateHandler struct{}

func (h *AddIceCandidateHandler) Transform(e event.RawEvent) interface{} {
	// addIceCandidate is typically an array with one candidate
	// Just return a count
	return ice.SimpleSummary()
}

// IceTrickleHandler handles IceTrickle events (SFU-side)
type IceTrickleHandler struct{}

func (h *IceTrickleHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return ice.SimpleSummary()
	}

	result := make(map[string]interface{})

	// Keep peerType
	if pt, ok := payload["peerType"].(float64); ok {
		result["pt"] = int(pt)
	}

	// Keep sessionId (shortened)
	if sid, ok := payload["sessionId"].(string); ok {
		result["sid"] = shortenID(sid)
	}

	// Parse the iceCandidate JSON string
	if iceCandStr, ok := payload["iceCandidate"].(string); ok {
		var iceCand map[string]interface{}
		if err := json.Unmarshal([]byte(iceCandStr), &iceCand); err == nil {
			summary := ice.ParseCandidateFromPayload(iceCand)
			if !summary.IsEndOfCandidates() {
				c := summary.ToMap()
				if len(c) > 0 {
					result["c"] = c
				}
			} else {
				result["eoc"] = 1
			}
		}
	}

	return result
}

// shortenID truncates UUIDs to first 8 chars
func shortenID(id string) string {
	if len(id) > 12 {
		return id[:4] + ".." + id[len(id)-4:]
	}
	return id
}
