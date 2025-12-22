package models

import "encoding/json"
// BoundingBox represents element position and dimensions
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// DeepgramWord represents a single word from speech-to-text
type DeepgramWord struct {
	Word            string  `json:"word"`
	Start           float64 `json:"start"`
	End             float64 `json:"end"`
	Confidence      float64 `json:"confidence"`
	PunctuatedWord  string  `json:"punctuated_word"`
}

// DeepgramResult contains the speech transcription
type DeepgramResult struct {
	Words []DeepgramWord `json:"words"`
}

// DomEvent represents a user interaction with the DOM
type DomEvent struct {
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Target    map[string]interface{} `json:"target"`
	Bounds    *BoundingBox           `json:"bounds,omitempty"`
}

// ProcessingRequest is the main input payload
type ProcessingRequest struct {
	SessionID            string          `json:"sessionId"`
	RecordingStartTimeMs int64           `json:"recordingStartTimeMs"`
	VideoDurationSec     float64         `json:"videoDurationSec"`
	DeepgramResponse     *DeepgramResult `json:"deepgramResponse"`
	DomEvents            []DomEvent      `json:"domEvents"`
}

// RawProcessingRequest allows sending the raw JSON from files directly
type RawProcessingRequest struct {
	DeepgramRaw json.RawMessage `json:"deepgramRaw"`
	DomRaw      json.RawMessage `json:"domRaw"`
}
