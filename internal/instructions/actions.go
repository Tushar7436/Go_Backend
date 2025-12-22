package instructions

import (
	"errors"
	"strings"

	"godemo/internal/models"
)

const (
	MaxInstructionGapSec = 60.0 // sanity guard
)

// GenerateActionInstructions converts normalized DOM actions into frontend-consumable replay instructions
func GenerateActionInstructions(
	actions []models.TimelineItem,
	videoDuration float64,
) ([]models.ActionInstruction, error) {

	if videoDuration <= 0 {
		return nil, errors.New("invalid video duration")
	}

	var instructions []models.ActionInstruction
	var lastT float64 = -1

	for _, a := range actions {
		if a.Kind != "action" {
			continue
		}

		// Time validation
		t := clampTime(a.T, videoDuration)

		if lastT >= 0 && (t-lastT) > MaxInstructionGapSec {
			// Large unexplained gap → still allowed, but frontend can handle silence
		}
		lastT = t

		inst := models.ActionInstruction{
			Timestamp: t,
			Action:    normalizeAction(a.Action),
			Selector:  extractSelector(a.Target),
			Bounds:    a.Bounds,
			Metadata:  extractMetadata(a),
			Effects:   suggestEffects(a),
		}

		// Drop instructions that are meaningless
		if inst.Action == "" {
			continue
		}

		instructions = append(instructions, inst)
	}

	return instructions, nil
}

// clampTime ensures time is within valid range
func clampTime(t float64, max float64) float64 {
	if t < 0 {
		return 0
	}
	if t > max {
		return max
	}
	return t
}

// normalizeAction standardizes action types to prevent frontend branching chaos
func normalizeAction(a string) string {
	switch strings.ToLower(a) {
	case "click":
		return "click"
	case "input", "type":
		return "input"
	case "scroll":
		return "scroll"
	case "hover":
		return "hover"
	case "zoom":
		return "zoom"
	case "navigation", "route_change":
		return "navigate"
	default:
		return ""
	}
}

// extractSelector extracts a frontend-friendly selector from target metadata
func extractSelector(target map[string]interface{}) string {
	if target == nil {
		return ""
	}

	keys := []string{
		"selector",
		"cssSelector",
		"id",
		"name",
	}

	for _, k := range keys {
		if v, ok := target[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}

	return ""
}

// extractMetadata extracts optional, non-rendering hints from the action
func extractMetadata(a models.TimelineItem) map[string]interface{} {
	meta := map[string]interface{}{}

	if a.Target != nil {
		if v, ok := a.Target["ariaLabel"]; ok {
			meta["ariaLabel"] = v
		}
		if v, ok := a.Target["text"]; ok {
			meta["text"] = v
		}
	}

	if len(meta) == 0 {
		return nil
	}

	return meta
}

// suggestEffects recommends effect types for an action (references only, not actual effect geometry)
func suggestEffects(a models.TimelineItem) []string {
	switch a.Action {
	case "click":
		return []string{"highlight", "zoom"}
	case "input":
		return []string{"focus"}
	case "navigation":
		return []string{"label"}
	default:
		return nil
	}
}
