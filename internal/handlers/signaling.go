package handlers

import (
	"encoding/json"

	"rtcstats/internal/event"
	"rtcstats/internal/transform"
)

// SignalingStateHandler handles signalingstatechange events
type SignalingStateHandler struct{}

func (h *SignalingStateHandler) Transform(e event.RawEvent) interface{} {
	var state string
	if err := json.Unmarshal(e.Payload, &state); err != nil {
		return nil
	}
	return transform.CompressSignalingState(state)
}

// ICEGatheringStateHandler handles icegatheringstatechange events
type ICEGatheringStateHandler struct{}

func (h *ICEGatheringStateHandler) Transform(e event.RawEvent) interface{} {
	var state string
	if err := json.Unmarshal(e.Payload, &state); err != nil {
		return nil
	}
	return transform.CompressICEGatheringState(state)
}

// ICEConnectionStateHandler handles iceconnectionstatechange events
type ICEConnectionStateHandler struct{}

func (h *ICEConnectionStateHandler) Transform(e event.RawEvent) interface{} {
	var state string
	if err := json.Unmarshal(e.Payload, &state); err != nil {
		return nil
	}
	return transform.CompressICEConnectionState(state)
}

// ConnectionStateHandler handles connectionstatechange events
type ConnectionStateHandler struct{}

func (h *ConnectionStateHandler) Transform(e event.RawEvent) interface{} {
	var state string
	if err := json.Unmarshal(e.Payload, &state); err != nil {
		return nil
	}
	return transform.CompressConnectionState(state)
}
