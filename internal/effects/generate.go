package effects

import (
	"math"

	"godemo/internal/models"
)

const (
	EffectCoverageRatio = 0.75 // effect lasts for 75% of window
	MinEffectDuration   = 0.4  // seconds
	MaxZoomScale        = 1.15
	MinZoomScale        = 1.08
)

// GenerateEffects creates visual display effects derived from
// DOM actions and narration windows.
func GenerateEffects(
	actions []models.TimelineItem,
	windows []models.Window,
	videoDuration float64,
) []models.DisplayEffect {

	var effects []models.DisplayEffect

	for _, w := range windows {

		if w.End <= w.Start {
			continue
		}

		action := findPrimaryActionForWindow(actions, w)
		if action == nil {
			continue
		}

		e := generateEffectForAction(*action, w, videoDuration)
		if e != nil {
			effects = append(effects, *e)
		}
	}

	return effects
}

// findPrimaryActionForWindow selects the primary action for a narration window.
// Rules:
// - First action at or after window start
// - Scroll is ignored unless it's the only action
// - Missing bounds → no effect
func findPrimaryActionForWindow(
	actions []models.TimelineItem,
	w models.Window,
) *models.TimelineItem {

	for _, a := range actions {
		if a.Kind != "action" {
			continue
		}

		if a.T < w.Start || a.T > w.End {
			continue
		}

		// Skip scroll as focal action
		if a.Action == "scroll" {
			continue
		}

		return &a
	}

	return nil
}

// generateEffectForAction creates the appropriate effect for an action type.
func generateEffectForAction(
	action models.TimelineItem,
	w models.Window,
	videoDuration float64,
) *models.DisplayEffect {

	start := w.Start
	end := start + math.Max(
		MinEffectDuration,
		(w.End-w.Start)*EffectCoverageRatio,
	)

	if end > w.End {
		end = w.End
	}
	if end > videoDuration {
		end = videoDuration
	}

	switch action.Action {

	case "click":
		return effectForClick(action, start, end)

	case "input":
		return effectForInput(action, start, end)

	case "navigation":
		return effectForNavigation(action, start, end)

	case "hover":
		return effectForHover(action, start, end)

	default:
		return nil
	}
}

// RULE 1: Click → Highlight + Light Zoom
// Highlights the clicked element and slightly zooms in for focus.
func effectForClick(
	action models.TimelineItem,
	start, end float64,
) *models.DisplayEffect {

	if action.Bounds == nil {
		return nil
	}

	return &models.DisplayEffect{
		Start: start,
		End:   end,
		Type:  "highlight",
		Target: &models.EffectTarget{
			Selector: extractSelector(action),
			Bounds:   action.Bounds,
		},
		Style: map[string]interface{}{
			"outline":       "glow",
			"dimBackground": true,
			"zoom": map[string]interface{}{
				"enabled": true,
				"scale":   MinZoomScale,
			},
		},
	}
}

// RULE 4: Input/Text Entry → Focus Box
// Creates a focus rectangle around input fields without zoom.
func effectForInput(
	action models.TimelineItem,
	start, end float64,
) *models.DisplayEffect {

	if action.Bounds == nil {
		return nil
	}

	return &models.DisplayEffect{
		Start: start,
		End:   end,
		Type:  "focus",
		Target: &models.EffectTarget{
			Selector: extractSelector(action),
			Bounds:   action.Bounds,
		},
		Style: map[string]interface{}{
			"borderColor":   "#0066ff",
			"borderWidth":   3,
			"dimBackground": true,
		},
	}
}

// RULE 2: Navigation → Label Overlay
// Creates a text label for navigation events without geometric effects.
func effectForNavigation(
	action models.TimelineItem,
	start, end float64,
) *models.DisplayEffect {

	label := extractLabel(action)
	if label == "" {
		return nil
	}

	return &models.DisplayEffect{
		Start: start,
		End:   end,
		Type:  "label",
		Style: map[string]interface{}{
			"text":      label,
			"position":  "top-center",
			"fontSize":  "14px",
			"fontColor": "#ffffff",
		},
	}
}

// effectForHover creates a soft highlight for hover events.
// Shorter duration than click since hover is transient.
func effectForHover(
	action models.TimelineItem,
	start, end float64,
) *models.DisplayEffect {

	if action.Bounds == nil {
		return nil
	}

	return &models.DisplayEffect{
		Start: start,
		End:   end,
		Type:  "highlight",
		Target: &models.EffectTarget{
			Selector: extractSelector(action),
			Bounds:   action.Bounds,
		},
		Style: map[string]interface{}{
			"outline": "soft",
			"opacity": 0.7,
		},
	}
}

// extractSelector retrieves the CSS selector from a target's metadata.
// Priority: selector > cssSelector > id > name
func extractSelector(action models.TimelineItem) string {
	if action.Target == nil {
		return ""
	}

	keys := []string{"selector", "cssSelector", "id", "name"}
	for _, k := range keys {
		if v, ok := action.Target[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// extractLabel retrieves human-readable label from target metadata.
// Priority: text > ariaLabel > label > name
func extractLabel(action models.TimelineItem) string {
	if action.Target == nil {
		return ""
	}

	keys := []string{"text", "ariaLabel", "label", "name"}
	for _, k := range keys {
		if v, ok := action.Target[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
