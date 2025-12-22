package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"godemo/internal/audio"
	"godemo/internal/effects"
	"godemo/internal/instructions"
	"godemo/internal/llm"
	"godemo/internal/models"
	"godemo/internal/normalize"
	"godemo/internal/timeline"
	"godemo/internal/windows"
	"os"
	"strings"
	"bytes"
	"io"
	"log"
)

// ProcessRecording handles the main video processing endpoint
func ProcessRecording(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Peek at the request to see if it's raw or pre-structured
	var rawData models.RawProcessingRequest
	bodyBytes, _ := io.ReadAll(r.Body)
	
	// Create a new reader since we consumed the body
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req models.ProcessingRequest
	
	// Attempt to parse as RAW first
	if err := json.Unmarshal(bodyBytes, &rawData); err == nil && len(rawData.DeepgramRaw) > 0 {
		log.Println("[INFO] Detected RAW request format, transforming...")
		transformed, err := transformRawRequest(rawData)
		if err != nil {
			http.Error(w, fmt.Sprintf("transform error: %v", err), http.StatusBadRequest)
			return
		}
		req = *transformed
	} else {
		// Fallback to standard structured request
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			http.Error(w, "invalid JSON structure", http.StatusBadRequest)
			return
		}
	}

	if req.VideoDurationSec <= 0 {
		http.Error(w, "videoDurationSec required (checking Deepgram metadata.duration)", http.StatusBadRequest)
		return
	}

	// 1. Normalize DOM events to timeline actions
	actions, err := normalize.NormalizeDomEvents(
		req.DomEvents,
		req.RecordingStartTimeMs,
		req.VideoDurationSec,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 2. Build canonical timeline (Speech + Actions)
	tl := timeline.BuildTimeline(req.DeepgramResponse, actions)

	// 3. Use LLM to refine script (if API key is available)
	geminiKey := os.Getenv("GEMINI_API_KEY")
	var narrations []models.Narration

	if geminiKey == "" {
		http.Error(w, "GEMINI_API_KEY environment variable is required", http.StatusInternalServerError)
		return
	}

	if req.DeepgramResponse != nil {
		// Extract full transcript
		var transcriptWords []string
		for _, word := range req.DeepgramResponse.Words {
			if word.PunctuatedWord != "" {
				transcriptWords = append(transcriptWords, word.PunctuatedWord)
			} else {
				transcriptWords = append(transcriptWords, word.Word)
			}
		}
		fullTranscript := strings.Join(transcriptWords, " ")

		// Call LLM for refinement
		refinedSegments, err := llm.RefineScript(llm.RefineScriptRequest{
			RawTranscript: fullTranscript,
			DOMEvents:     req.DomEvents,
			VideoDuration: req.VideoDurationSec,
			DeepgramWords: req.DeepgramResponse.Words,
		}, geminiKey)

		if err != nil {
			http.Error(w, fmt.Sprintf("Gemini API error: %v", err), http.StatusInternalServerError)
			return
		}

		if len(refinedSegments) == 0 {
			http.Error(w, "Gemini returned no narration segments", http.StatusInternalServerError)
			return
		}

		// Convert LLM segments to Narration format
		for i, seg := range refinedSegments {
			narrations = append(narrations, models.Narration{
				WindowIndex: i,
				Start:       seg.Start,
				End:         seg.End,
				Text:        seg.Text,
				MusicStyle:  seg.MusicStyle,
			})
		}
	}

	// 4. Generate Replay Instructions & Effects
	win := windows.ExtractNarrationWindows(tl, req.VideoDurationSec)
	replayInst, _ := instructions.GenerateActionInstructions(actions, req.VideoDurationSec)
	fx := effects.GenerateEffects(actions, win, req.VideoDurationSec)

	// 5. Generate Audio
	var audioFile string
	if len(narrations) > 0 {
		// Create simple windows from narration segments
		var narrationWindows []models.Window
		for _, n := range narrations {
			narrationWindows = append(narrationWindows, models.Window{
				Start: n.Start,
				End:   n.End,
			})
		}

		chunks, err := audio.MapNarrationsToAudioChunks(narrations, narrationWindows, audio.TTSDeepgram)
		if err == nil {
			audioFile = req.SessionID + ".mp3"
			audio.SaveFullAudio(chunks, audio.TTSDeepgram, audioFile, req.VideoDurationSec)
		}
	}

	// 6. Response Construction
	resp := map[string]interface{}{
		"sessionId":      req.SessionID,
		"videoDuration":  req.VideoDurationSec,
		"narrations":     narrations,
		"instructions":   replayInst,
		"displayEffects": fx,
		"audioFile":      "/audio/" + audioFile,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
