package handlers

import (
	"encoding/json"
	"strings"

	"rtcstats/internal/event"
	"rtcstats/internal/transform"
)

// EnumerateDevicesHandler handles navigator.mediaDevices.enumerateDevices
type EnumerateDevicesHandler struct{}

func (h *EnumerateDevicesHandler) Transform(e event.RawEvent) interface{} {
	var devices []map[string]interface{}
	if err := json.Unmarshal(e.Payload, &devices); err != nil {
		return nil
	}

	// Count devices by kind and check for labels
	counts := map[string]int{
		"ai": 0, // audioinput
		"vi": 0, // videoinput
		"ao": 0, // audiooutput
	}
	hasLabel := false

	for _, dev := range devices {
		if kind, ok := dev["kind"].(string); ok {
			shortKind := transform.CompressDeviceKind(kind)
			counts[shortKind]++
		}
		if label, ok := dev["label"].(string); ok && label != "" {
			hasLabel = true
		}
	}

	result := make(map[string]interface{})
	for k, v := range counts {
		if v > 0 {
			result[k] = v
		}
	}
	result["hl"] = transform.BoolToInt(hasLabel)

	return result
}

// GetUserMediaHandler handles getUserMedia.* events
type GetUserMediaHandler struct{}

func (h *GetUserMediaHandler) Transform(e event.RawEvent) interface{} {
	// Check if this is OnSuccess, OnFailure, or the request itself
	if strings.HasSuffix(e.Name, ".OnSuccess") {
		return h.handleSuccess(e)
	}
	if strings.HasSuffix(e.Name, ".OnFailure") {
		return h.handleFailure(e)
	}
	return h.handleRequest(e)
}

func (h *GetUserMediaHandler) handleRequest(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	// Check for audio constraints
	if audio, ok := payload["audio"]; ok {
		if audioBool, ok := audio.(bool); ok && audioBool {
			result["a"] = 1
		} else if audioMap, ok := audio.(map[string]interface{}); ok {
			result["a"] = 1
			// Extract audio processing flags
			if agc, ok := audioMap["autoGainControl"].(bool); ok && agc {
				result["agc"] = 1
			}
			if ns, ok := audioMap["noiseSuppression"].(bool); ok && ns {
				result["ns"] = 1
			}
			if ec, ok := audioMap["echoCancellation"].(bool); ok && ec {
				result["ec"] = 1
			}
		}
	}

	// Check for video constraints
	if video, ok := payload["video"]; ok {
		if videoBool, ok := video.(bool); ok && videoBool {
			result["v"] = 1
		} else if videoMap, ok := video.(map[string]interface{}); ok {
			result["v"] = 1
			// Extract video dimensions
			if w, ok := videoMap["width"].(float64); ok {
				result["w"] = int(w)
			}
			if h, ok := videoMap["height"].(float64); ok {
				result["h"] = int(h)
			}
		}
	}

	return result
}

func (h *GetUserMediaHandler) handleSuccess(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	// Count tracks by kind
	if tracks, ok := payload["tracks"].([]interface{}); ok {
		for _, t := range tracks {
			if track, ok := t.(map[string]interface{}); ok {
				if kind, ok := track["kind"].(string); ok {
					shortKind := transform.CompressMediaKind(kind)
					result[shortKind] = 1
				}
			}
		}
	}

	return result
}

func (h *GetUserMediaHandler) handleFailure(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return map[string]interface{}{"ok": 0}
	}

	result := map[string]interface{}{"ok": 0}

	if name, ok := payload["name"].(string); ok {
		result["errc"] = name
	}
	if msg, ok := payload["message"].(string); ok {
		if len(msg) > 50 {
			msg = msg[:50] + "..."
		}
		result["err"] = msg
	}

	return result
}

// PermissionsHandler handles permissions.query(*) events
type PermissionsHandler struct{}

func (h *PermissionsHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		// Try as string (state directly)
		var state string
		if err := json.Unmarshal(e.Payload, &state); err == nil {
			return map[string]interface{}{"st": transform.CompressPermissionState(state)}
		}
		return nil
	}

	if state, ok := payload["state"].(string); ok {
		return map[string]interface{}{"st": transform.CompressPermissionState(state)}
	}

	return nil
}

// SetSinkIdHandler handles navigator.mediaDevices.setSinkId
type SetSinkIdHandler struct{}

func (h *SetSinkIdHandler) Transform(e event.RawEvent) interface{} {
	// The sink ID is usually a hash already, just mark success
	return map[string]interface{}{"ok": 1}
}
