package transform

// FieldMap maps original field names to short versions
var FieldMap = map[string]string{
	"deviceId":         "did",
	"groupId":          "gid",
	"sessionId":        "sid",
	"unifiedSessionId": "usid",
	"userId":           "uid",
	"user_id":          "uid",
	"kind":             "k",
	"trackType":        "tt",
	"track_type":       "tt",
	"width":            "w",
	"height":           "h",
	"direction":        "dir",
	"enabled":          "en",
	"muted":            "mu",
	"readyState":       "rs",
	"peerType":         "pt",
	"sdpMLineIndex":    "mli",
	"sdpMid":           "mid",
}

// DropFields lists fields to remove entirely
var DropFields = map[string]bool{
	"label":            true,
	"timestamp":        true,
	"sdp":              true,
	"candidate":        true,
	"iceCandidate":     true,
	"usernameFragment": true,
}

// RenameField returns the short name for a field, or the original if no mapping exists
func RenameField(name string) string {
	if short, ok := FieldMap[name]; ok {
		return short
	}
	return name
}

// ShouldDropField returns true if the field should be removed
func ShouldDropField(name string) bool {
	return DropFields[name]
}

// RenameMapKeys renames keys in a map according to FieldMap
func RenameMapKeys(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		if ShouldDropField(k) {
			continue
		}
		newKey := RenameField(k)

		// Recursively handle nested maps
		if nested, ok := v.(map[string]interface{}); ok {
			v = RenameMapKeys(nested)
		}

		result[newKey] = v
	}
	return result
}
