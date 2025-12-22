package normalize

import (
	"errors"

	"godemo/internal/models"
)

// NormalizeDomEvents converts raw DOM events to normalized timeline items with validation
func NormalizeDomEvents(
	events []models.DomEvent,
	startMs int64,
	videoDuration float64,
) ([]models.TimelineItem, error) {

	if videoDuration > 3600 {
		return nil, errors.New("video duration too large")
	}

	var out []models.TimelineItem
	var lastScrollTime float64 = -100.0

	for _, e := range events {
		// 1. Ignore noise
		if e.Type == "" || e.Type == "dom_mutation" {
			continue
		}

		if e.Timestamp == 0 {
			continue
		}

		t := float64(e.Timestamp-startMs) / 1000
		if t < 0 { t = 0 }
		if t > videoDuration { t = videoDuration }

		// 2. Collapse consecutive scrolls (debounce)
		// If we get multiple scrolls within 1.5 seconds, treat them as one continuous action
		if e.Type == "scroll" {
			if t-lastScrollTime < 1.5 {
				continue 
			}
			lastScrollTime = t
		}

		out = append(out, models.TimelineItem{
			T:      t,
			Kind:   "action",
			Action: e.Type,
			Target: e.Target,
			Bounds: e.Bounds,
		})
	}

	return out, nil
}
