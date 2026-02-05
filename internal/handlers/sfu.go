package handlers

import (
	"encoding/json"
	"strings"

	"rtcstats/internal/event"
	"rtcstats/internal/sdp"
	"rtcstats/internal/transform"
)

// SignalWSOpenHandler handles signal.ws.open events
type SignalWSOpenHandler struct{}

func (h *SignalWSOpenHandler) Transform(e event.RawEvent) interface{} {
	// Usually just {isTrusted: true} - compress to simple ok
	return map[string]interface{}{"ok": 1}
}

// JoinRequestHandler handles joinRequest events
type JoinRequestHandler struct{}

func (h *JoinRequestHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	// Navigate through requestPayload wrapper
	reqPayload := payload
	if rp, ok := payload["requestPayload"].(map[string]interface{}); ok {
		if jr, ok := rp["joinRequest"].(map[string]interface{}); ok {
			reqPayload = jr
		}
	}

	// Session ID
	if sid, ok := reqPayload["sessionId"].(string); ok {
		result["sid"] = shortenID(sid)
	}

	// Fast reconnect
	if fr, ok := reqPayload["fastReconnect"].(bool); ok && fr {
		result["fr"] = 1
	}

	// Capabilities
	if caps, ok := reqPayload["capabilities"].([]interface{}); ok && len(caps) > 0 {
		result["cap"] = caps
	}

	// Client details
	if cd, ok := reqPayload["clientDetails"].(map[string]interface{}); ok {
		// SDK info
		if sdkInfo, ok := cd["sdk"].(map[string]interface{}); ok {
			sdkType := int(0)
			if t, ok := sdkInfo["type"].(float64); ok {
				sdkType = int(t)
			}
			version := ""
			if major, ok := sdkInfo["major"].(string); ok {
				version = major
				if minor, ok := sdkInfo["minor"].(string); ok {
					version += "." + minor
					if patch, ok := sdkInfo["patch"].(string); ok {
						version += "." + patch
					}
				}
			}
			if version != "" {
				result["sdk"] = []interface{}{sdkType, version}
			}
		}

		// OS info
		if osInfo, ok := cd["os"].(map[string]interface{}); ok {
			osArr := make([]string, 0, 3)
			if name, ok := osInfo["name"].(string); ok {
				osArr = append(osArr, strings.ToLower(name[:min(3, len(name))]))
			}
			if ver, ok := osInfo["version"].(string); ok {
				osArr = append(osArr, ver)
			}
			if arch, ok := osInfo["architecture"].(string); ok {
				osArr = append(osArr, arch)
			}
			if len(osArr) > 0 {
				result["os"] = osArr
			}
		}

		// Browser info
		if brInfo, ok := cd["browser"].(map[string]interface{}); ok {
			brArr := make([]string, 0, 2)
			if name, ok := brInfo["name"].(string); ok {
				brArr = append(brArr, strings.ToLower(name[:min(2, len(name))]))
			}
			if ver, ok := brInfo["version"].(string); ok {
				// Just major version
				if idx := strings.Index(ver, "."); idx > 0 {
					ver = ver[:idx]
				}
				brArr = append(brArr, ver)
			}
			if len(brArr) > 0 {
				result["br"] = brArr
			}
		}
	}

	// SDP summaries
	if pubSdp, ok := reqPayload["publisherSdp"].(string); ok {
		digest := sdp.CreateSDPDigest(pubSdp, "offer")
		if digest != nil {
			result["pub_sdp_sum"] = digest
		}
	}
	if subSdp, ok := reqPayload["subscriberSdp"].(string); ok {
		digest := sdp.CreateSDPDigest(subSdp, "offer")
		if digest != nil {
			result["sub_sdp_sum"] = digest
		}
	}

	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SetPublisherHandler handles SetPublisher events
type SetPublisherHandler struct{}

func (h *SetPublisherHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	// Session ID
	if sid, ok := payload["sessionId"].(string); ok {
		result["sid"] = shortenID(sid)
	}

	// SDP summary
	if sdpStr, ok := payload["sdp"].(string); ok {
		digest := sdp.CreateSDPDigest(sdpStr, "offer")
		if digest != nil {
			result["sdp_sum"] = digest
		}
	}

	// Track summary
	if tracks, ok := payload["tracks"].([]interface{}); ok {
		tr := make([]map[string]interface{}, 0, len(tracks))
		for _, t := range tracks {
			track, ok := t.(map[string]interface{})
			if !ok {
				continue
			}

			trackSum := make(map[string]interface{})

			if mid, ok := track["mid"].(string); ok {
				trackSum["mid"] = mid
			}

			if tt, ok := track["trackType"].(float64); ok {
				trackSum["tt"] = int(tt)
			}

			// Codec name
			if codec, ok := track["codec"].(map[string]interface{}); ok {
				if name, ok := codec["name"].(string); ok {
					trackSum["c"] = strings.ToLower(name)
				}
			}

			// Simulcast layers
			if layers, ok := track["layers"].([]interface{}); ok && len(layers) > 0 {
				sc := make([][]interface{}, 0, len(layers))
				for _, l := range layers {
					layer, ok := l.(map[string]interface{})
					if !ok {
						continue
					}
					layerArr := make([]interface{}, 0, 4)
					if rid, ok := layer["rid"].(string); ok {
						layerArr = append(layerArr, rid)
					}
					if br, ok := layer["bitrate"].(float64); ok {
						layerArr = append(layerArr, int(br/1000)) // kbps
					}
					if vd, ok := layer["videoDimension"].(map[string]interface{}); ok {
						if w, ok := vd["width"].(float64); ok {
							layerArr = append(layerArr, int(w))
						}
						if h, ok := vd["height"].(float64); ok {
							layerArr = append(layerArr, int(h))
						}
					}
					if len(layerArr) > 0 {
						sc = append(sc, layerArr)
					}
				}
				if len(sc) > 0 {
					trackSum["sc"] = sc
				}
			}

			if len(trackSum) > 0 {
				tr = append(tr, trackSum)
			}
		}
		if len(tr) > 0 {
			result["tr"] = tr
		}
	}

	return result
}

// SetPublisherResponseHandler handles SetPublisherResponse events
type SetPublisherResponseHandler struct{}

func (h *SetPublisherResponseHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	// SDP summary
	if sdpStr, ok := payload["sdp"].(string); ok {
		digest := sdp.CreateSDPDigest(sdpStr, "answer")
		if digest != nil {
			result["sdp_sum"] = digest
		}
	}

	return result
}

// SendAnswerHandler handles SendAnswer events
type SendAnswerHandler struct{}

func (h *SendAnswerHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	// Session ID
	if sid, ok := payload["sessionId"].(string); ok {
		result["sid"] = shortenID(sid)
	}

	// SDP summary
	if sdpStr, ok := payload["sdp"].(string); ok {
		digest := sdp.CreateSDPDigest(sdpStr, "answer")
		if digest != nil {
			result["sdp_sum"] = digest
		}
	}

	return result
}

// UpdateMuteStatesHandler handles UpdateMuteStates events
type UpdateMuteStatesHandler struct{}

func (h *UpdateMuteStatesHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	if states, ok := payload["muteStates"].([]interface{}); ok {
		mu := make(map[string]int)
		for _, s := range states {
			state, ok := s.(map[string]interface{})
			if !ok {
				continue
			}
			tt := transform.CompressTrackType(state["trackType"])
			muted := 0
			if m, ok := state["muted"].(bool); ok && m {
				muted = 1
			}
			if tt == 1 {
				mu["a"] = muted
			} else if tt == 2 {
				mu["v"] = muted
			}
		}
		if len(mu) > 0 {
			result["mu"] = mu
		}
	}

	return result
}

// UpdateSubscriptionsHandler handles UpdateSubscriptions events
type UpdateSubscriptionsHandler struct{}

func (h *UpdateSubscriptionsHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	if sid, ok := payload["sessionId"].(string); ok {
		result["sid"] = shortenID(sid)
	}

	if tracks, ok := payload["tracks"].([]interface{}); ok {
		tr := make([]map[string]interface{}, 0, len(tracks))
		for _, t := range tracks {
			track, ok := t.(map[string]interface{})
			if !ok {
				continue
			}

			trackSum := make(map[string]interface{})

			if uid, ok := track["userId"].(string); ok {
				trackSum["u"] = uid
			}

			if tt, ok := track["trackType"].(float64); ok {
				trackSum["tt"] = int(tt)
			}

			if dim, ok := track["dimension"].(map[string]interface{}); ok {
				wh := make([]int, 0, 2)
				if w, ok := dim["width"].(float64); ok {
					wh = append(wh, int(w))
				}
				if h, ok := dim["height"].(float64); ok {
					wh = append(wh, int(h))
				}
				if len(wh) == 2 {
					trackSum["wh"] = wh
				}
			}

			if len(trackSum) > 0 {
				tr = append(tr, trackSum)
			}
		}
		if len(tr) > 0 {
			result["tr"] = tr
		}
	}

	return result
}

// ConnectionQualityHandler handles connectionQualityChanged events
type ConnectionQualityHandler struct{}

func (h *ConnectionQualityHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		// Try as direct value
		var q float64
		if err := json.Unmarshal(e.Payload, &q); err == nil {
			return map[string]interface{}{"q": int(q)}
		}
		return nil
	}

	if q, ok := payload["quality"].(float64); ok {
		return map[string]interface{}{"q": int(q)}
	}

	return nil
}

// TrackMappingHandler handles sfu.track.mapping events
type TrackMappingHandler struct{}

func (h *TrackMappingHandler) Transform(e event.RawEvent) interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	result := make(map[string]interface{})

	// Direction
	if dir, ok := payload["direction"].(string); ok {
		if dir == "inbound" {
			result["dir"] = "in"
		} else if dir == "outbound" {
			result["dir"] = "out"
		}
	}

	// Track type
	if tt, ok := payload["track_type"].(string); ok {
		result["tt"] = transform.CompressTrackType(tt)
	}

	// Codec (normalize)
	if codec, ok := payload["codec"].(string); ok {
		// Remove trailing colon and params
		codec = strings.TrimSuffix(codec, ":")
		if idx := strings.Index(codec, ":"); idx > 0 {
			codec = codec[:idx]
		}
		result["c"] = strings.ToLower(codec)
	}

	// User ID from participant
	if part, ok := payload["participant"].(map[string]interface{}); ok {
		if uid, ok := part["user_id"].(string); ok {
			result["uid"] = uid
		}
	}

	// SSRC (keep as number, it's useful for correlation)
	if ssrc, ok := payload["ssrc"].(float64); ok {
		result["s"] = int64(ssrc)
	}

	return result
}
