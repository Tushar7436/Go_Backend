package validate

import (
	"errors"
	"fmt"

	"godemo/internal/models"
)

const (
	MaxAllowedDriftSec = 0.15 // hard sync tolerance
)

// ValidateFinalOutput performs final sync and consistency checks before returning to frontend
func ValidateFinalOutput(
	videoDuration float64,
	windows []models.Window,
	narrations []models.Narration,
	audioChunks []models.AudioChunk,
	actions []models.ActionInstruction,
	effects []models.DisplayEffect,
) error {

	if videoDuration <= 0 {
		return errors.New("invalid video duration")
	}

	if err := validateWindows(videoDuration, windows); err != nil {
		return err
	}

	if err := validateNarrations(windows, narrations); err != nil {
		return err
	}

	if err := validateAudioSync(windows, audioChunks); err != nil {
		return err
	}

	if err := validateActions(videoDuration, actions); err != nil {
		return err
	}

	if err := validateEffects(videoDuration, effects); err != nil {
		return err
	}

	return nil
}

// validateWindows ensures narration windows are valid and non-overlapping
func validateWindows(videoDuration float64, windows []models.Window) error {

	lastEnd := 0.0

	for i, w := range windows {

		if w.Start < 0 || w.End < 0 {
			return fmt.Errorf("window %d has negative time", i)
		}

		if w.Start >= w.End {
			return fmt.Errorf("window %d has invalid range", i)
		}

		if w.End > videoDuration {
			return fmt.Errorf("window %d exceeds video duration", i)
		}

		if w.Start < lastEnd {
			return fmt.Errorf("window %d overlaps previous window", i)
		}

		lastEnd = w.End
	}

	return nil
}

// validateNarrations ensures narration stays within its assigned window
func validateNarrations(
	windows []models.Window,
	narrations []models.Narration,
) error {

	for _, n := range narrations {

		if n.Text == "" {
			continue // silence is valid
		}

		if n.WindowIndex < 0 || n.WindowIndex >= len(windows) {
			return fmt.Errorf("narration references invalid window %d", n.WindowIndex)
		}

		w := windows[n.WindowIndex]

		if n.Start < w.Start || n.End > w.End {
			return fmt.Errorf(
				"narration out of window bounds (%.2f–%.2f vs %.2f–%.2f)",
				n.Start, n.End, w.Start, w.End,
			)
		}
	}

	return nil
}

// validateAudioSync ensures audio never outlives its narration window
func validateAudioSync(
	windows []models.Window,
	audioChunks []models.AudioChunk,
) error {

	for _, a := range audioChunks {

		if a.WindowIndex < 0 || a.WindowIndex >= len(windows) {
			return fmt.Errorf("audio chunk references invalid window %d", a.WindowIndex)
		}

		w := windows[a.WindowIndex]
		windowDur := w.End - w.Start

		if a.Duration <= 0 {
			return fmt.Errorf("audio duration must be positive")
		}

		if a.Duration > windowDur+MaxAllowedDriftSec {
			return fmt.Errorf(
				"audio duration %.2fs exceeds window %.2fs",
				a.Duration, windowDur,
			)
		}
	}

	return nil
}

// validateActions ensures action instructions have valid timestamps and geometry
func validateActions(
	videoDuration float64,
	actions []models.ActionInstruction,
) error {

	for i, a := range actions {

		if a.Timestamp < 0 || a.Timestamp > videoDuration {
			return fmt.Errorf("action %d has invalid timestamp %.2f", i, a.Timestamp)
		}

		if a.Action == "" {
			return fmt.Errorf("action %d has empty action type", i)
		}

		// Bounds are optional, but if present must be sane
		if a.Bounds != nil {
			if a.Bounds.Width <= 0 || a.Bounds.Height <= 0 {
				return fmt.Errorf("action %d has invalid bounds", i)
			}
		}
	}

	return nil
}

// validateEffects ensures display effects don't overflow video bounds and have valid geometry
func validateEffects(
	videoDuration float64,
	effects []models.DisplayEffect,
) error {

	for i, e := range effects {

		if e.Start < 0 || e.End < 0 {
			return fmt.Errorf("effect %d has negative time", i)
		}

		if e.Start >= e.End {
			return fmt.Errorf("effect %d has invalid range", i)
		}

		if e.End > videoDuration {
			return fmt.Errorf("effect %d exceeds video duration", i)
		}

		if e.Type == "" {
			return fmt.Errorf("effect %d missing type", i)
		}

		if e.Target != nil && e.Target.Bounds != nil {
			if e.Target.Bounds.Width <= 0 || e.Target.Bounds.Height <= 0 {
				return fmt.Errorf("effect %d has invalid bounds", i)
			}
		}
	}

	return nil
}
