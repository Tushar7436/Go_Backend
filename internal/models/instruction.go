package models

// ActionInstruction is a frontend-consumable replay instruction for a DOM action
type ActionInstruction struct {
	Timestamp float64                `json:"t"`
	Action    string                 `json:"action"`
	Selector  string                 `json:"selector,omitempty"`
	Bounds    *BoundingBox           `json:"bounds,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Effects   []string               `json:"effects,omitempty"`
}
