package event

import "encoding/json"

// RawEvent represents the input format: [eventName, scope, payload, ts]
type RawEvent struct {
	Name    string
	Scope   *string         // nullable
	Payload json.RawMessage // raw JSON for flexible handling
	TS      int64
}

// CompressedEvent represents the output format
type CompressedEvent struct {
	Name    string      `json:"n"`
	Scope   string      `json:"s,omitempty"`
	Payload interface{} `json:"p,omitempty"`
	TS      int64       `json:"ts,omitempty"`
	DT      *int64      `json:"dt,omitempty"`
}

// TimestampMode controls how timestamps are output
type TimestampMode int

const (
	TSAbsolute TimestampMode = iota
	TSDelta
	TSBoth
)
