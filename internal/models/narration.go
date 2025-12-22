package models

// Narration represents generated speech text for a narration window
type Narration struct {
	WindowIndex int     `json:"windowIndex"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Text        string  `json:"text"`
	MusicStyle  string  `json:"musicStyle,omitempty"` // e.g., "tech", "upbeat", "travel"
}

// AudioChunk represents a piece of synthesized audio mapped to a narration window
type AudioChunk struct {
	WindowIndex int     `json:"windowIndex"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Duration    float64 `json:"duration"`
	Text        string  `json:"text"`
	Provider    string  `json:"provider"`
	MusicStyle  string  `json:"musicStyle,omitempty"`
	AudioURL    string  `json:"audioUrl,omitempty"`
	AudioBytes  []byte  `json:"-"`
}
