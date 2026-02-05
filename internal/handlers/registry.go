package handlers

import (
	"rtcstats/internal/event"
	"strings"
)

// Handler transforms a RawEvent into a CompressedEvent payload
type Handler interface {
	Transform(e event.RawEvent) interface{}
}

// HandlerFunc is a function type that implements Handler
type HandlerFunc func(e event.RawEvent) interface{}

func (f HandlerFunc) Transform(e event.RawEvent) interface{} {
	return f(e)
}

// Registry maps event names to handlers
type Registry struct {
	exact    map[string]Handler
	prefix   map[string]Handler
	suffix   map[string]Handler
	fallback Handler
}

// NewRegistry creates a new handler registry with all handlers registered
func NewRegistry() *Registry {
	r := &Registry{
		exact:    make(map[string]Handler),
		prefix:   make(map[string]Handler),
		suffix:   make(map[string]Handler),
		fallback: &GenericHandler{},
	}

	// Register all handlers
	r.registerMediaDeviceHandlers()
	r.registerPeerConnectionHandlers()
	r.registerSignalingHandlers()
	r.registerICEHandlers()
	r.registerSFUHandlers()

	return r
}

func (r *Registry) registerMediaDeviceHandlers() {
	ed := &EnumerateDevicesHandler{}
	r.exact["navigator.mediaDevices.enumerateDevices"] = ed

	gum := &GetUserMediaHandler{}
	r.prefix["navigator.mediaDevices.getUserMedia."] = gum

	r.exact["navigator.mediaDevices.setSinkId"] = &SetSinkIdHandler{}
	r.exact["setUseWebAudio"] = &PassthroughHandler{}

	perm := &PermissionsHandler{}
	r.prefix["permissions.query"] = perm
}

func (r *Registry) registerPeerConnectionHandlers() {
	r.exact["create"] = &CreatePCHandler{}
	r.exact["negotiationneeded"] = &NullPayloadHandler{}

	offer := &CreateOfferHandler{}
	r.exact["createOffer"] = offer
	r.exact["createOfferOnSuccess"] = &CreateOfferSuccessHandler{}
	r.exact["createOfferOnFailure"] = &FailureHandler{}

	answer := &CreateAnswerHandler{}
	r.exact["createAnswer"] = answer
	r.exact["createAnswerOnSuccess"] = &CreateAnswerSuccessHandler{}
	r.exact["createAnswerOnFailure"] = &FailureHandler{}

	setLocal := &SetDescriptionHandler{IsLocal: true}
	r.exact["setLocalDescription"] = setLocal
	r.exact["setLocalDescriptionOnSuccess"] = &NullPayloadHandler{}
	r.exact["setLocalDescriptionOnFailure"] = &FailureHandler{}

	setRemote := &SetDescriptionHandler{IsLocal: false}
	r.exact["setRemoteDescription"] = setRemote
	r.exact["setRemoteDescriptionOnSuccess"] = &NullPayloadHandler{}
	r.exact["setRemoteDescriptionOnFailure"] = &FailureHandler{}

	r.exact["ontrack"] = &OnTrackHandler{}
	r.exact["getstats"] = &GetStatsHandler{}
}

func (r *Registry) registerSignalingHandlers() {
	r.exact["signalingstatechange"] = &SignalingStateHandler{}
	r.exact["icegatheringstatechange"] = &ICEGatheringStateHandler{}
	r.exact["iceconnectionstatechange"] = &ICEConnectionStateHandler{}
	r.exact["connectionstatechange"] = &ConnectionStateHandler{}
}

func (r *Registry) registerICEHandlers() {
	r.exact["onicecandidate"] = &OnIceCandidateHandler{}
	r.exact["addIceCandidate"] = &AddIceCandidateHandler{}
	r.exact["addIceCandidateOnSuccess"] = &NullPayloadHandler{}
	r.exact["addIceCandidateOnFailure"] = &FailureHandler{}
	r.exact["IceTrickle"] = &IceTrickleHandler{}
}

func (r *Registry) registerSFUHandlers() {
	r.exact["signal.ws.open"] = &SignalWSOpenHandler{}
	r.exact["joinRequest"] = &JoinRequestHandler{}
	r.exact["SetPublisher"] = &SetPublisherHandler{}
	r.exact["SetPublisherResponse"] = &SetPublisherResponseHandler{}
	r.exact["SendAnswer"] = &SendAnswerHandler{}
	r.exact["UpdateMuteStates"] = &UpdateMuteStatesHandler{}
	r.exact["UpdateSubscriptions"] = &UpdateSubscriptionsHandler{}
	r.exact["connectionQualityChanged"] = &ConnectionQualityHandler{}
	r.exact["sfu.track.mapping"] = &TrackMappingHandler{}
}

// Get returns the handler for an event name
func (r *Registry) Get(name string) Handler {
	// Try exact match first
	if h, ok := r.exact[name]; ok {
		return h
	}

	// Try prefix matches
	for prefix, h := range r.prefix {
		if strings.HasPrefix(name, prefix) {
			return h
		}
	}

	// Try suffix matches
	for suffix, h := range r.suffix {
		if strings.HasSuffix(name, suffix) {
			return h
		}
	}

	return r.fallback
}

// Register adds a handler for an exact event name match
func (r *Registry) Register(name string, h Handler) {
	r.exact[name] = h
}

// RegisterPrefix adds a handler for events matching a prefix
func (r *Registry) RegisterPrefix(prefix string, h Handler) {
	r.prefix[prefix] = h
}

// RegisterSuffix adds a handler for events matching a suffix
func (r *Registry) RegisterSuffix(suffix string, h Handler) {
	r.suffix[suffix] = h
}
