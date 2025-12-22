package script

import (
	"errors"
	"fmt"
	"strings"

	"godemo/internal/models"
)

const (
	MinWindowDurationSec = 0.8   // below this → silence
	MaxWordsPerSecond    = 2.5   // conservative narration speed
	MinWordConfidence    = 0.50  // Deepgram confidence cutoff (lowered to preserve more words)
	MaxSummaryActions    = 3     // max actions to narrate in one window
)

// GenerateScript creates narration text for a single narration window.
// It is deterministic, action-grounded, and time-safe.
func GenerateScript(
	window models.Window,
	timeline []models.TimelineItem,
) (string, error) {

	if window.End <= window.Start {
		return "", errors.New("invalid narration window")
	}

	windowDuration := window.End - window.Start

	// Rule 1: very short windows → silence
	if windowDuration < MinWindowDurationSec {
		return "", nil
	}

	// STEP 1: Collect high-confidence speech in this window
	speech := extractSpeechForWindow(window, timeline)

	// STEP 2: If there's speech, use it (user's actual words take priority)
	if speech != "" {
		return speech, nil
	}

	// STEP 3: No speech - collect actions and generate description
	actions := extractActionsForWindow(window, timeline)
	
	if len(actions) == 0 {
		return "", nil // Empty window, stay silent
	}

	// Check if all actions are just scrolls
	allScrolls := true
	for _, a := range actions {
		if a.Action != "scroll" {
			allScrolls = false
			break
		}
	}
	
	// Skip narration for scroll-only windows unless it's sustained scrolling
	if allScrolls && windowDuration < 3.0 {
		return "", nil
	}
	
	narration := narrateActions(actions)

	// STEP 4: Enforce duration constraint
	maxWords := int(windowDuration * MaxWordsPerSecond)
	
	if len(strings.Fields(narration)) > maxWords {
		if maxWords < 4 {
			return "", nil // Too short for meaningful sentence
		}
		narration = trimToWordLimit(narration, maxWords)
	}

	if len(strings.Fields(narration)) < 3 {
		return "", nil // Minimum 3 words
	}

	return narration, nil
}

// extractActionsForWindow collects actions that occur in this window
func extractActionsForWindow(
	window models.Window,
	timeline []models.TimelineItem,
) []models.TimelineItem {

	var actions []models.TimelineItem

	for _, item := range timeline {
		if item.Kind != "action" {
			continue
		}

		// Action inside or immediately after window start
		// CRITICAL: We use < window.End (exclusive) so we don't count the action 
		// that defines the start of the NEXT window.
		if item.T >= window.Start && item.T < window.End {
			actions = append(actions, item)
		}
	}

	if len(actions) > MaxSummaryActions {
		return actions[:MaxSummaryActions]
	}

	return actions
}

// extractSpeechForWindow extracts high-confidence words from speech
func extractSpeechForWindow(
	window models.Window,
	timeline []models.TimelineItem,
) string {

	var words []string

	for _, item := range timeline {
		if item.Kind != "speech_word" {
			continue
		}

		if item.T < window.Start || item.T > window.End {
			continue
		}

		if item.Confidence < MinWordConfidence {
			continue
		}

		if isFillerWord(item.Word) {
			continue
		}

		// Use punctuated word if available, otherwise use raw word
		word := item.PunctuatedWord
		if word == "" {
			word = item.Word
		}
		words = append(words, word)
	}

	if len(words) == 0 {
		return ""
	}

	// Return as-is without extra normalization (Deepgram handles it)
	return strings.Join(words, " ")
}

// narrateActions generates narration from actions (action-first logic)
func narrateActions(actions []models.TimelineItem) string {

	if len(actions) == 0 {
		return ""
	}

	// Priority 1: If there is a Click or Input, narrate that specifically
	// even if there are scrolls in the same window.
	for _, a := range actions {
		if a.Action == "click" || a.Action == "input" {
			return narrateSingleAction(a)
		}
	}

	// Priority 2: Single action
	if len(actions) == 1 {
		return narrateSingleAction(actions[0])
	}

	// Priority 3: Multiple actions (summarized)
	return narrateMultipleActions(actions)
}

// narrateSingleAction creates narration for a single action
func narrateSingleAction(action models.TimelineItem) string {

	switch action.Action {

	case "click":
		label := extractTargetLabel(action)
		if label != "" {
			return fmt.Sprintf("The %s is selected.", label)
		}
		return "An item is selected."

	case "input":
		label := extractTargetLabel(action)
		if label != "" {
			return fmt.Sprintf("Text is entered into the %s field.", label)
		}
		return "Text is entered into a field."

	case "navigation":
		label := extractTargetLabel(action)
		if label != "" {
			return fmt.Sprintf("The user navigates to %s.", label)
		}
		return "The user navigates to a new section."

	case "scroll":
		return "The page is scrolled to explore more content."

	default:
		return ""
	}
}

// narrateMultipleActions creates summary narration for multiple actions
func narrateMultipleActions(actions []models.TimelineItem) string {
	counts := make(map[string]int)
	for _, a := range actions {
		counts[a.Action]++
	}

	if counts["scroll"] > 1 && len(counts) == 1 {
		return "The user scrolls through the page to explore more content."
	}

	return "Various sections of the interface are explored."
}

// extractTargetLabel gets a user-friendly label from the target DOM element
func extractTargetLabel(action models.TimelineItem) string {

	if action.Target == nil {
		return ""
	}

	// Preferred order of label sources
	keys := []string{"ariaLabel", "text", "label", "name"}

	for _, k := range keys {
		if v, ok := action.Target[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				// IGNORE if the label is too long (indicates container noise)
				if len(s) > 40 {
					continue
				}
				return normalizeLabel(s)
			}
		}
	}

	return ""
}

// trimToWordLimit truncates text to fit within word budget
func trimToWordLimit(text string, maxWords int) string {
	if maxWords <= 0 {
		return ""
	}

	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}

	return strings.Join(words[:maxWords], " ")
}

// isFillerWord filters out filler words from speech
func isFillerWord(w string) bool {
	switch strings.ToLower(w) {
	case "uh", "um", "ah", "er", "hmm":
		return true
	default:
		return false
	}
}

// normalizeSentence applies basic grammar normalization
func normalizeSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	// Capitalize first letter
	if len(s) > 0 {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}

// normalizeLabel converts DOM labels to narration format
func normalizeLabel(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return s
}
