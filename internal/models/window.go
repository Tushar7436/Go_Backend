package models

// Window represents a narration window between actions
type Window struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}
