package handlers

import (
	"encoding/json"

	"rtcstats/internal/event"
	"rtcstats/internal/transform"
)

// GenericHandler is the fallback handler that applies basic transformations
type GenericHandler struct{}

func (h *GenericHandler) Transform(e event.RawEvent) interface{} {
	if len(e.Payload) == 0 || string(e.Payload) == "null" {
		return nil
	}

	var payload interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	// Apply transformations based on payload type
	switch p := payload.(type) {
	case map[string]interface{}:
		p = transform.StripSecrets(p)
		p = transform.RenameMapKeys(p)
		return p

	case []interface{}:
		// For arrays, transform each element if it's a map
		for i, elem := range p {
			if m, ok := elem.(map[string]interface{}); ok {
				m = transform.StripSecrets(m)
				m = transform.RenameMapKeys(m)
				p[i] = m
			}
		}
		return p

	default:
		return payload
	}
}

// NullPayloadHandler returns nil payload (for events where payload is irrelevant)
type NullPayloadHandler struct{}

func (h *NullPayloadHandler) Transform(e event.RawEvent) interface{} {
	return nil
}

// PassthroughHandler passes the payload through with minimal changes
type PassthroughHandler struct{}

func (h *PassthroughHandler) Transform(e event.RawEvent) interface{} {
	if len(e.Payload) == 0 || string(e.Payload) == "null" {
		return nil
	}

	var payload interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}
	return payload
}

// FailureHandler handles OnFailure events
type FailureHandler struct{}

func (h *FailureHandler) Transform(e event.RawEvent) interface{} {
	if len(e.Payload) == 0 || string(e.Payload) == "null" {
		return map[string]interface{}{"ok": 0}
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return map[string]interface{}{"ok": 0}
	}

	result := map[string]interface{}{"ok": 0}

	// Extract error info
	if errName, ok := payload["name"].(string); ok {
		result["errc"] = errName
	}
	if errMsg, ok := payload["message"].(string); ok {
		// Truncate long messages
		if len(errMsg) > 100 {
			errMsg = errMsg[:100] + "..."
		}
		result["err"] = errMsg
	}

	return result
}
