package timeline

import (
	"sort"

	"godemo/internal/models"
)

// BuildTimeline merges speech words and actions into a single canonical timeline
func BuildTimeline(
	deepgram *models.DeepgramResult,
	actions []models.TimelineItem,
) []models.TimelineItem {

	var tl []models.TimelineItem

	if deepgram != nil {
		for _, w := range deepgram.Words {
			tl = append(tl, models.TimelineItem{
				T:              w.Start,
				Kind:           "speech_word",
				Word:           w.Word,
				PunctuatedWord: w.PunctuatedWord,
				Confidence:     w.Confidence,
			})
		}
	}

	tl = append(tl, actions...)

	sort.Slice(tl, func(i, j int) bool {
		return tl[i].T < tl[j].T
	})

	return tl
}
