package windows

import "godemo/internal/models"

// ExtractNarrationWindows finds gaps between actions where narration can occur
func ExtractNarrationWindows(
	timeline []models.TimelineItem,
	videoDuration float64,
) []models.Window {

	// Strategy: Group speech into natural sentences based on timing gaps
	// A "sentence" ends when there's a pause > 0.8 seconds between words
	// Actions don't break speech - only natural pauses do

	var windows []models.Window
	
	type speechSegment struct {
		start float64
		end   float64
		words []models.TimelineItem
	}
	
	var segments []speechSegment
	var currentSegment *speechSegment
	
	// Group speech words into natural segments
	for _, item := range timeline {
		if item.Kind != "speech_word" {
			continue
		}
		
		if currentSegment == nil {
			// Start new segment
			currentSegment = &speechSegment{
				start: item.T,
				end:   item.T,
				words: []models.TimelineItem{item},
			}
		} else {
			// Check gap from last word
			gap := item.T - currentSegment.end
			
			if gap > 0.8 {
				// Natural pause - end current segment
				segments = append(segments, *currentSegment)
				currentSegment = &speechSegment{
					start: item.T,
					end:   item.T,
					words: []models.TimelineItem{item},
				}
			} else {
				// Continue current segment
				currentSegment.end = item.T
				currentSegment.words = append(currentSegment.words, item)
			}
		}
	}
	
	// Close final segment
	if currentSegment != nil {
		segments = append(segments, *currentSegment)
	}
	
	// Convert segments to windows
	cursor := 0.0
	
	for _, seg := range segments {
		// Fill gap before speech with action-based narration
		if seg.start > cursor && (seg.start - cursor) > 0.5 {
			windows = append(windows, models.Window{
				Start: cursor,
				End:   seg.start,
			})
		}
		
		// Add speech window (with small buffer for word completion)
		windows = append(windows, models.Window{
			Start: seg.start,
			End:   seg.end + 0.3,
		})
		
		cursor = seg.end + 0.3
	}
	
	// Final action window if time remains
	if cursor < videoDuration && (videoDuration - cursor) > 0.5 {
		windows = append(windows, models.Window{
			Start: cursor,
			End:   videoDuration,
		})
	}

	return windows
}
