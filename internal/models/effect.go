package models

// EffectTarget specifies what the effect applies to
type EffectTarget struct {
	Selector string       `json:"selector,omitempty"`
	Bounds   *BoundingBox `json:"bounds,omitempty"`
}

// DisplayEffect represents a visual effect to be applied during playback
type DisplayEffect struct {
	Start  float64            `json:"start"`
	End    float64            `json:"end"`
	Type   string             `json:"type"` // "highlight" | "zoom" | "focus" | "dim" | "blur" | "label"
	Target *EffectTarget      `json:"target,omitempty"`
	Style  map[string]interface{} `json:"style,omitempty"`
}
