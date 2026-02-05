package transform

// MediaKind maps media kinds to short codes
var MediaKind = map[string]string{
	"audio": "a",
	"video": "v",
}

// DeviceKind maps device kinds to short codes
var DeviceKind = map[string]string{
	"audioinput":  "ai",
	"videoinput":  "vi",
	"audiooutput": "ao",
	"videoinput2": "vi", // rare duplicate
}

// SignalingState maps signaling states to integers
var SignalingState = map[string]int{
	"stable":             0,
	"have-local-offer":   1,
	"have-remote-offer":  2,
	"have-local-pranswer": 3,
	"have-remote-pranswer": 4,
	"closed":             5,
}

// ICEConnectionState maps ICE connection states to integers
var ICEConnectionState = map[string]int{
	"new":          0,
	"checking":     1,
	"connected":    2,
	"completed":    3,
	"failed":       4,
	"disconnected": 5,
	"closed":       6,
}

// ICEGatheringState maps ICE gathering states to integers
var ICEGatheringState = map[string]int{
	"new":       0,
	"gathering": 1,
	"complete":  2,
}

// ConnectionState maps connection states to integers
var ConnectionState = map[string]int{
	"new":          0,
	"connecting":   1,
	"connected":    2,
	"disconnected": 3,
	"failed":       4,
	"closed":       5,
}

// PermissionState maps permission states to short codes
var PermissionState = map[string]string{
	"granted": "g",
	"prompt":  "p",
	"denied":  "d",
}

// TrackType maps track type strings/numbers to integers
var TrackType = map[string]int{
	"TRACK_TYPE_UNSPECIFIED": 0,
	"TRACK_TYPE_AUDIO":       1,
	"TRACK_TYPE_VIDEO":       2,
	"audio":                  1,
	"video":                  2,
}

// BoolToInt converts a boolean to 0 or 1
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// CompressMediaKind returns short code for media kind
func CompressMediaKind(kind string) string {
	if short, ok := MediaKind[kind]; ok {
		return short
	}
	return kind
}

// CompressDeviceKind returns short code for device kind
func CompressDeviceKind(kind string) string {
	if short, ok := DeviceKind[kind]; ok {
		return short
	}
	return kind
}

// CompressSignalingState returns int for signaling state
func CompressSignalingState(state string) int {
	if i, ok := SignalingState[state]; ok {
		return i
	}
	return -1
}

// CompressICEConnectionState returns int for ICE connection state
func CompressICEConnectionState(state string) int {
	if i, ok := ICEConnectionState[state]; ok {
		return i
	}
	return -1
}

// CompressICEGatheringState returns int for ICE gathering state
func CompressICEGatheringState(state string) int {
	if i, ok := ICEGatheringState[state]; ok {
		return i
	}
	return -1
}

// CompressConnectionState returns int for connection state
func CompressConnectionState(state string) int {
	if i, ok := ConnectionState[state]; ok {
		return i
	}
	return -1
}

// CompressPermissionState returns short code for permission state
func CompressPermissionState(state string) string {
	if short, ok := PermissionState[state]; ok {
		return short
	}
	return state
}

// CompressTrackType returns int for track type
func CompressTrackType(tt interface{}) int {
	switch v := tt.(type) {
	case string:
		if i, ok := TrackType[v]; ok {
			return i
		}
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}
