package models

// TimelineItem represents a single event in the canonical timeline
type TimelineItem struct {
	T              float64                `json:"t"`
	Kind           string                 `json:"kind"` // "speech_word" | "action"
	Word           string                 `json:"word,omitempty"`
	PunctuatedWord string                 `json:"punctuatedWord,omitempty"`
	Confidence     float64                `json:"confidence,omitempty"`
	Action         string                 `json:"action,omitempty"`
	Target         map[string]interface{} `json:"target,omitempty"`
	Bounds         *BoundingBox           `json:"bounds,omitempty"`
}
