package handlers

import (
	"encoding/json"
	"math"
	"strings"

	"rtcstats/internal/event"
)

// reportType identifies the classification of an RTCStatsReport entry.
type reportType int

const (
	rtUnknown reportType = iota
	rtOutboundVideo
	rtOutboundAudio
	rtInboundAudio
	rtInboundVideo
	rtRemoteInbound
	rtCandidatePairActive
	rtCandidatePairRelay
	rtMediaSourceVideo
	rtConnectionQuality
)

// fieldSpec describes a single field to extract from a stats entry.
type fieldSpec struct {
	original  string // original WebRTC field name
	shortKey  string // compressed output key
	isCounter bool   // true = delta, false = gauge
}

// Per-report-type field specifications (order matches spec sections).

var outboundVideoFields = []fieldSpec{
	{"bytesSent", "bs", true},
	{"headerBytesSent", "hbs", true},
	{"packetsSent", "ps", true},
	{"framesEncoded", "fe", true},
	{"framesPerSecond", "fps", false},
	{"qpSum", "qp", true},
	{"totalEncodeTime", "tet", true},
	{"totalEncodedBytesTarget", "tebt", true},
	{"pliCount", "pli", true},
	{"hugeFramesSent", "hfs", true},
}

var outboundAudioFields = []fieldSpec{
	{"bytesSent", "bs", true},
	{"headerBytesSent", "hbs", true},
	{"packetsSent", "ps", true},
}

var inboundAudioFields = []fieldSpec{
	{"bytesReceived", "br", true},
	{"headerBytesReceived", "hbr", true},
	{"packetsReceived", "pr", true},
	{"jitter", "j", false},
	{"audioLevel", "al", false},
	{"totalAudioEnergy", "tae", true},
	{"totalSamplesDuration", "tsd", true},
	{"totalSamplesReceived", "tsr", true},
	{"concealedSamples", "cs", true},
	{"concealmentEvents", "ce", true},
	{"removedSamplesForAcceleration", "rsa", true},
	{"silentConcealedSamples", "scs", true},
	{"jitterBufferDelay", "jbd", true},
	{"jitterBufferEmittedCount", "jbe", true},
	{"jitterBufferMinimumDelay", "jbm", true},
	{"jitterBufferTargetDelay", "jbt", true},
}

var inboundVideoFields = []fieldSpec{
	{"bytesReceived", "br", true},
	{"headerBytesReceived", "hbr", true},
	{"packetsReceived", "pr", true},
	{"jitter", "j", false},
	{"framesDecoded", "fd", true},
	{"framesReceived", "fr", true},
	{"framesPerSecond", "fps", false},
	{"framesAssembledFromMultiplePackets", "fam", true},
	{"qpSum", "qp", true},
	{"totalDecodeTime", "tdt", true},
	{"totalInterFrameDelay", "tifd", true},
	{"totalSquaredInterFrameDelay", "tsid", true},
	{"totalAssemblyTime", "tat", true},
	{"totalProcessingDelay", "tpd", true},
	{"jitterBufferDelay", "jbd", true},
	{"jitterBufferEmittedCount", "jbe", true},
	{"jitterBufferMinimumDelay", "jbm", true},
	{"jitterBufferTargetDelay", "jbt", true},
	{"packetsLost", "pl", true},
	{"packetsDiscarded", "pd", true},
	{"nackCount", "nk", true},
	{"keyFramesDecoded", "kfd", true},
	{"freezeCount", "fzc", true},
	{"totalFreezesDuration", "fzd", true},
	{"framesDropped", "fdr", true},
}

var remoteInboundFields = []fieldSpec{
	{"roundTripTime", "rtt", false},
	{"jitter", "j", false},
	{"packetsReceived", "pr", true},
	{"totalRoundTripTime", "trtt", true},
	{"roundTripTimeMeasurements", "rttm", true},
}

var candidatePairActiveFields = []fieldSpec{
	{"bytesSent", "bs", true},
	{"bytesReceived", "br", true},
	{"currentRoundTripTime", "rtt", false},
	{"responsesReceived", "rr", true},
	{"totalRoundTripTime", "trtt", true},
}

var candidatePairRelayFields = []fieldSpec{
	{"bytesSent", "bs", true},
	{"packetsSent", "ps", true},
	{"remoteTimestamp", "rts", false},
}

var cqFields = []fieldSpec{
	{"score", "s", false},
	{"avgScore", "as", false},
	{"mosScore", "mos", false},
}

var mediaSourceVideoFields = []fieldSpec{
	{"frames", "f", true},
	{"framesPerSecond", "fps", false},
}

// Fields to always drop from entries (before classification).
var globalDropFields = map[string]bool{
	"timestamp":                    true,
	"remoteId":                     true,
	"localId":                      true,
	"codecId":                      true,
	"ssrc":                         true,
	"mediaType":                    true,
	"kind":                         true,
	"type":                         true,
	"framesSent":                   true,
	"frameHeight":                  true,
	"frameWidth":                   true,
	"lastPacketReceivedTimestamp":  true,
	"lastPacketSentTimestamp":      true,
	"discardedPackets":             true,
	"fractionLost":                 true,
}

// StatsSnapshot captures raw values from a getstats sample for later
// recomputation of deltas against a different baseline.
type StatsSnapshot struct {
	Scope     string
	RawValues map[string]map[string]float64 // stateKey → field → raw value
}

// GetStatsHandler compresses RTCStatsReport data per the spec.
// It holds state for delta computation across samples.
type GetStatsHandler struct {
	prevValues        map[string]map[string]float64 // key: "scope:entryID" → field→value
	lastEmittedValues map[string]map[string]float64 // baseline for emission recomputation
}

func (h *GetStatsHandler) Transform(e event.RawEvent) interface{} {
	if h.prevValues == nil {
		h.prevValues = make(map[string]map[string]float64)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}

	scope := ""
	if e.Scope != nil {
		scope = *e.Scope
	}

	// Output buckets
	var outV []map[string]interface{}
	var outA map[string]interface{}
	var inA map[string]interface{}
	var inV map[string]interface{}
	var rttArr []map[string]interface{}
	var cpArr []map[string]interface{}
	var cq map[string]interface{}
	var ms map[string]interface{}

	for entryID, raw := range payload {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			// Top-level non-dict value (e.g. "timestamp") → skip
			continue
		}

		rt := classifyEntry(entryID, entry)
		if rt == rtUnknown {
			continue
		}

		stateKey := scope + ":" + entryID
		fields := fieldsForType(rt)
		compressed := h.compressEntry(stateKey, entry, fields)
		if len(compressed) == 0 {
			continue
		}

		switch rt {
		case rtOutboundVideo:
			outV = append(outV, compressed)
		case rtOutboundAudio:
			outA = compressed
		case rtInboundAudio:
			inA = compressed
		case rtInboundVideo:
			inV = compressed
		case rtRemoteInbound:
			rttArr = append(rttArr, compressed)
		case rtCandidatePairActive:
			cpArr = append(cpArr, compressed)
		case rtCandidatePairRelay:
			cpArr = append(cpArr, compressed)
		case rtConnectionQuality:
			cq = compressed
		case rtMediaSourceVideo:
			ms = compressed
		}
	}

	result := make(map[string]interface{})
	if len(outV) > 0 {
		result["out_v"] = outV
	}
	if outA != nil {
		result["out_a"] = outA
	}
	if inA != nil {
		result["in_a"] = inA
	}
	if inV != nil {
		result["in_v"] = inV
	}
	if len(rttArr) > 0 {
		result["rtt"] = rttArr
	}
	if len(cpArr) > 0 {
		result["cp"] = cpArr
	}
	if cq != nil {
		result["cq"] = cq
	}
	if ms != nil {
		result["ms"] = ms
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// ExtractAndTransform works like Transform but also returns a StatsSnapshot
// containing the raw values for each entry. The snapshot can be used later
// to recompute deltas against a different baseline (lastEmittedValues).
func (h *GetStatsHandler) ExtractAndTransform(e event.RawEvent) (interface{}, *StatsSnapshot) {
	if h.prevValues == nil {
		h.prevValues = make(map[string]map[string]float64)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil, nil
	}

	scope := ""
	if e.Scope != nil {
		scope = *e.Scope
	}

	snapshot := &StatsSnapshot{
		Scope:     scope,
		RawValues: make(map[string]map[string]float64),
	}

	var outV []map[string]interface{}
	var outA map[string]interface{}
	var inA map[string]interface{}
	var inV map[string]interface{}
	var rttArr []map[string]interface{}
	var cpArr []map[string]interface{}
	var cq map[string]interface{}
	var ms map[string]interface{}

	for entryID, raw := range payload {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		rt := classifyEntry(entryID, entry)
		if rt == rtUnknown {
			continue
		}

		stateKey := scope + ":" + entryID
		fields := fieldsForType(rt)

		// Capture raw values into snapshot
		rawVals := make(map[string]float64)
		for _, f := range fields {
			if v, ok := entry[f.original]; ok {
				if fv, ok := toFloat64(v); ok {
					rawVals[f.original] = fv
				}
			}
		}
		snapshot.RawValues[stateKey] = rawVals

		compressed := h.compressEntry(stateKey, entry, fields)
		if len(compressed) == 0 {
			continue
		}

		switch rt {
		case rtOutboundVideo:
			outV = append(outV, compressed)
		case rtOutboundAudio:
			outA = compressed
		case rtInboundAudio:
			inA = compressed
		case rtInboundVideo:
			inV = compressed
		case rtRemoteInbound:
			rttArr = append(rttArr, compressed)
		case rtCandidatePairActive:
			cpArr = append(cpArr, compressed)
		case rtCandidatePairRelay:
			cpArr = append(cpArr, compressed)
		case rtConnectionQuality:
			cq = compressed
		case rtMediaSourceVideo:
			ms = compressed
		}
	}

	result := make(map[string]interface{})
	if len(outV) > 0 {
		result["out_v"] = outV
	}
	if outA != nil {
		result["out_a"] = outA
	}
	if inA != nil {
		result["in_a"] = inA
	}
	if inV != nil {
		result["in_v"] = inV
	}
	if len(rttArr) > 0 {
		result["rtt"] = rttArr
	}
	if len(cpArr) > 0 {
		result["cp"] = cpArr
	}
	if cq != nil {
		result["cq"] = cq
	}
	if ms != nil {
		result["ms"] = ms
	}

	if len(result) == 0 {
		return nil, snapshot
	}
	return result, snapshot
}

// RecomputeForEmission recomputes the compressed output for a snapshot using
// lastEmittedValues as the baseline for counter deltas instead of prevValues.
// This produces correct accumulated deltas when samples have been skipped.
func (h *GetStatsHandler) RecomputeForEmission(snapshot *StatsSnapshot) interface{} {
	if h.lastEmittedValues == nil {
		h.lastEmittedValues = make(map[string]map[string]float64)
	}

	var outV []map[string]interface{}
	var outA map[string]interface{}
	var inA map[string]interface{}
	var inV map[string]interface{}
	var rttArr []map[string]interface{}
	var cpArr []map[string]interface{}
	var cq map[string]interface{}
	var ms map[string]interface{}

	for stateKey, rawVals := range snapshot.RawValues {
		// Extract entryID from stateKey (scope:entryID)
		entryID := stateKey
		if idx := strings.LastIndex(stateKey, ":"); idx >= 0 {
			entryID = stateKey[idx+1:]
		}

		// Reconstruct a fake entry to classify
		fakeEntry := make(map[string]interface{})
		for k, v := range rawVals {
			fakeEntry[k] = v
		}
		rt := classifyEntry(entryID, fakeEntry)
		if rt == rtUnknown {
			continue
		}

		fields := fieldsForType(rt)
		prev := h.lastEmittedValues[stateKey]
		compressed := make(map[string]interface{})

		for _, f := range fields {
			val, ok := rawVals[f.original]
			if !ok {
				continue
			}

			if f.isCounter {
				if prev != nil {
					if prevVal, hasPrev := prev[f.original]; hasPrev {
						delta := roundFloat(val-prevVal, 6)
						if delta != 0 {
							compressed[f.shortKey] = cleanNumber(delta)
						}
					} else {
						rounded := roundFloat(val, 6)
						if rounded != 0 {
							compressed[f.shortKey] = cleanNumber(rounded)
						}
					}
				} else {
					rounded := roundFloat(val, 6)
					if rounded != 0 {
						compressed[f.shortKey] = cleanNumber(rounded)
					}
				}
			} else {
				rounded := roundFloat(val, 6)
				if rounded != 0 {
					compressed[f.shortKey] = cleanNumber(rounded)
				}
			}
		}

		if len(compressed) == 0 {
			continue
		}

		switch rt {
		case rtOutboundVideo:
			outV = append(outV, compressed)
		case rtOutboundAudio:
			outA = compressed
		case rtInboundAudio:
			inA = compressed
		case rtInboundVideo:
			inV = compressed
		case rtRemoteInbound:
			rttArr = append(rttArr, compressed)
		case rtCandidatePairActive:
			cpArr = append(cpArr, compressed)
		case rtCandidatePairRelay:
			cpArr = append(cpArr, compressed)
		case rtConnectionQuality:
			cq = compressed
		case rtMediaSourceVideo:
			ms = compressed
		}
	}

	result := make(map[string]interface{})
	if len(outV) > 0 {
		result["out_v"] = outV
	}
	if outA != nil {
		result["out_a"] = outA
	}
	if inA != nil {
		result["in_a"] = inA
	}
	if inV != nil {
		result["in_v"] = inV
	}
	if len(rttArr) > 0 {
		result["rtt"] = rttArr
	}
	if len(cpArr) > 0 {
		result["cp"] = cpArr
	}
	if cq != nil {
		result["cq"] = cq
	}
	if ms != nil {
		result["ms"] = ms
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// UpdateEmittedBaseline copies snapshot raw values into lastEmittedValues.
// Call this after a sample has been successfully emitted.
func (h *GetStatsHandler) UpdateEmittedBaseline(snapshot *StatsSnapshot) {
	if h.lastEmittedValues == nil {
		h.lastEmittedValues = make(map[string]map[string]float64)
	}
	for stateKey, rawVals := range snapshot.RawValues {
		cp := make(map[string]float64, len(rawVals))
		for k, v := range rawVals {
			cp[k] = v
		}
		h.lastEmittedValues[stateKey] = cp
	}
}

// classifyEntry determines the report type of a stats entry by field fingerprint.
// Priority is applied top-to-bottom per the spec.
func classifyEntry(entryID string, entry map[string]interface{}) reportType {
	// CQ: key is literally "CQ"
	if entryID == "CQ" {
		return rtConnectionQuality
	}

	// Media source video: ID matches mediasource_video_*
	if strings.HasPrefix(entryID, "mediasource_video_") {
		if _, ok := entry["frames"]; ok {
			return rtMediaSourceVideo
		}
	}

	// Media source audio: always drop
	if strings.HasPrefix(entryID, "mediasource_audio_") {
		return rtUnknown
	}

	// Timestamp-only: only field is "timestamp"
	if isTimestampOnly(entry) {
		return rtUnknown
	}

	// Outbound video: has framesEncoded + bytesSent
	_, hasFE := entry["framesEncoded"]
	_, hasBS := entry["bytesSent"]
	_, hasHBS := entry["headerBytesSent"]
	_, hasBR := entry["bytesReceived"]
	_, hasFD := entry["framesDecoded"]
	_, hasRTT := entry["roundTripTime"]
	_, hasRTTM := entry["roundTripTimeMeasurements"]
	_, hasRR := entry["responsesReceived"]
	_, hasCRTT := entry["currentRoundTripTime"]
	_, hasRTS := entry["remoteTimestamp"]
	_, hasTAE := entry["totalAudioEnergy"]
	_, hasAL := entry["audioLevel"]

	if hasFE && hasBS {
		return rtOutboundVideo
	}

	// Outbound audio: has bytesSent + headerBytesSent, no framesEncoded
	if hasBS && hasHBS && !hasFE {
		// Distinguish from candidate pairs: candidate pairs have remoteTimestamp or responsesReceived/currentRoundTripTime
		if !hasRTS && !hasRR && !hasCRTT && !hasBR {
			return rtOutboundAudio
		}
	}

	// Inbound audio: has bytesReceived + (totalAudioEnergy or audioLevel)
	if hasBR && (hasTAE || hasAL) {
		return rtInboundAudio
	}

	// Inbound video: has bytesReceived + framesDecoded
	if hasBR && hasFD {
		return rtInboundVideo
	}

	// Remote-inbound RTP: has roundTripTime or roundTripTimeMeasurements, no responsesReceived/currentRoundTripTime
	if (hasRTT || hasRTTM) && !hasRR && !hasCRTT {
		return rtRemoteInbound
	}

	// Candidate pair active: has responsesReceived or currentRoundTripTime
	if hasRR || hasCRTT {
		return rtCandidatePairActive
	}

	// Candidate pair relay: has bytesSent + remoteTimestamp, no responsesReceived
	if hasBS && hasRTS && !hasRR {
		return rtCandidatePairRelay
	}

	return rtUnknown
}

// isTimestampOnly returns true if the only field in entry is "timestamp".
func isTimestampOnly(entry map[string]interface{}) bool {
	if len(entry) == 0 {
		return true
	}
	if len(entry) == 1 {
		_, ok := entry["timestamp"]
		return ok
	}
	return false
}

// fieldsForType returns the field specs for a given report type.
func fieldsForType(rt reportType) []fieldSpec {
	switch rt {
	case rtOutboundVideo:
		return outboundVideoFields
	case rtOutboundAudio:
		return outboundAudioFields
	case rtInboundAudio:
		return inboundAudioFields
	case rtInboundVideo:
		return inboundVideoFields
	case rtRemoteInbound:
		return remoteInboundFields
	case rtCandidatePairActive:
		return candidatePairActiveFields
	case rtCandidatePairRelay:
		return candidatePairRelayFields
	case rtConnectionQuality:
		return cqFields
	case rtMediaSourceVideo:
		return mediaSourceVideoFields
	default:
		return nil
	}
}

// compressEntry extracts and compresses fields from a stats entry.
// For counters, it computes deltas against previous values stored under stateKey.
func (h *GetStatsHandler) compressEntry(stateKey string, entry map[string]interface{}, fields []fieldSpec) map[string]interface{} {
	prev := h.prevValues[stateKey]
	curr := make(map[string]float64)
	result := make(map[string]interface{})

	for _, f := range fields {
		raw, ok := entry[f.original]
		if !ok {
			continue
		}

		val, ok := toFloat64(raw)
		if !ok {
			continue
		}

		if f.isCounter {
			curr[f.original] = val
			if prev != nil {
				if prevVal, hasPrev := prev[f.original]; hasPrev {
					delta := val - prevVal
					delta = roundFloat(delta, 6)
					if delta != 0 {
						result[f.shortKey] = cleanNumber(delta)
					}
				} else {
					// First time seeing this field for this entry
					rounded := roundFloat(val, 6)
					if rounded != 0 {
						result[f.shortKey] = cleanNumber(rounded)
					}
				}
			} else {
				// First sample for this entry — emit absolute
				rounded := roundFloat(val, 6)
				if rounded != 0 {
					result[f.shortKey] = cleanNumber(rounded)
				}
			}
		} else {
			// Gauge: keep as-is, omit if zero
			rounded := roundFloat(val, 6)
			if rounded != 0 {
				result[f.shortKey] = cleanNumber(rounded)
			}
		}
	}

	// Update stored previous values
	if prev == nil {
		h.prevValues[stateKey] = curr
	} else {
		for k, v := range curr {
			prev[k] = v
		}
	}

	return result
}

// toFloat64 converts a JSON-decoded numeric value to float64.
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// roundFloat rounds a float to n decimal places.
func roundFloat(val float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(val*pow) / pow
}

// cleanNumber returns an int if the float has no fractional part, otherwise the float.
// This keeps JSON output clean (e.g. 298 instead of 298.0).
func cleanNumber(val float64) interface{} {
	if val == math.Trunc(val) && val >= math.MinInt64 && val <= math.MaxInt64 {
		return int64(val)
	}
	return val
}
