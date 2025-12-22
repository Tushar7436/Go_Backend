package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"godemo/internal/models"
)

const (
	GeminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent"
)

// RefineScriptRequest contains all inputs for LLM script refinement
type RefineScriptRequest struct {
	RawTranscript string
	DOMEvents     []models.DomEvent
	VideoDuration float64
	DeepgramWords []models.DeepgramWord
}

// RefinedSegment represents a clean narration segment with timing
type RefinedSegment struct {
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Text       string  `json:"text"`
	MusicStyle string  `json:"musicStyle"` // Suggestion for background music (e.g. tech, upbeat, travel)
}

// RefineScript uses Gemini to clean up the transcript and generate professional narration
func RefineScript(req RefineScriptRequest, apiKey string) ([]RefinedSegment, error) {
	log.Println("[STEP 1] RefineScript called")
	
	if apiKey == "" {
		return nil, errors.New("gemini API key required")
	}
	log.Printf("[STEP 2] API key present: %d chars", len(apiKey))

	// Build the prompt
	prompt := buildPrompt(req)
	log.Printf("[STEP 3] Built prompt, length: %d chars", len(prompt))

	// Call Gemini API
	segments, err := callGeminiAPI(prompt, apiKey)
	if err != nil {
		log.Printf("[ERROR] Gemini API call failed: %v", err)
		return nil, fmt.Errorf("gemini API error: %w", err)
	}

	log.Printf("[STEP 4] Successfully got %d segments from Gemini", len(segments))
	return segments, nil
}

func buildPrompt(req RefineScriptRequest) string {
	// Extract action summary
	actionSummary := summarizeActions(req.DOMEvents)

	prompt := fmt.Sprintf(`You are a professional video narrator. Your task is to create a clean, natural narration script for a screen recording.

VIDEO DETAILS:
- Duration: %.2f seconds
- Topic: E-commerce website demonstration

RAW TRANSCRIPT (messy):
%s

USER ACTIONS:
%s

TASK:
1. Clean up the transcript and create a professional, HYPER-ENTHUSIASTIC, ENCOURAGING, and BOLD script.
   
2. IMPORTANT - BE INFECTIOUSLY EXCITED:
   - Use high-energy, persuasive language. Use words like "Revolutionary!", "Incredible!", "Absolute game-changer!", "You're going to love this!".
   - Make the user feel ENCOURAGED to shop and explore. 
   - Every sentence should sound like it's a huge benefit for the viewer.

3. FILL THE SILENCE:
   - The video is %.2f seconds long. You MUST provide enough words to fill the entire duration.
   - For every segment, aim for ~2.5 words per second of duration. 
   - DESCRIBE the interface with passion if you run out of transcript text.

4. EXPRESSIVE PUNCTUATION:
   - Aggressively use exclamation marks (!) and rhetorical questions (?) to keep the energy peaking!

5. STRETCH NARRATION:
   - Back-to-back segments (no gaps). 0-10, 10-20, etc.

6. BACKGROUND MUSIC THEMES:
   - Suggest "upbeat", "tech", "luxury", "travel", or "minimal".

7. Return JSON array format:
[
  {"start": 0, "end": 10.0, "text": "Welcome to Flipkart! The absolute BEST place for all your shopping needs! You've got to see these incredible brands and deals we have for you today! Let's jump in!", "musicStyle": "upbeat"}
]

IMPORTANT RULES:
- Use VIBRANT, BOLD, and INFECTIOUSLY ENTHUSIASTIC language.
- Provide a "musicStyle" for EVERY segment.
- FILL THE TIME with high-energy hype.
- Return ONLY valid JSON.

OUTPUT (JSON only):`, req.VideoDuration, req.RawTranscript, actionSummary, req.VideoDuration, req.VideoDuration)

	return prompt
}

func summarizeActions(events []models.DomEvent) string {
	if len(events) == 0 {
		return "No specific actions recorded"
	}

	var actions []string
	clickCount := 0
	scrollCount := 0

	for _, e := range events {
		switch e.Type {
		case "click":
			clickCount++
			if clickCount <= 3 {
				selector := "element"
				if target, ok := e.Target["selector"].(string); ok && target != "" {
					selector = target
				}
				actions = append(actions, fmt.Sprintf("- Clicked: %s", selector))
			}
		case "scroll":
			scrollCount++
		case "input":
			actions = append(actions, "- Entered text into form field")
		case "navigation":
			actions = append(actions, "- Navigated to new page")
		}
	}

	if scrollCount > 0 {
		actions = append(actions, fmt.Sprintf("- Scrolled through page (%d times)", scrollCount))
	}

	return strings.Join(actions, "\n")
}

func callGeminiAPI(prompt string, apiKey string) ([]RefinedSegment, error) {
	log.Println("[API-1] Building Gemini request payload")
	
	// Build request payload with higher maxOutputTokens
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.3,
			"maxOutputTokens": 4096, // Increased from 2048
		},
	}

	jsonData, _ := json.Marshal(payload)
	log.Printf("[API-2] Request payload size: %d bytes", len(jsonData))

	// Make HTTP request
	url := fmt.Sprintf("%s?key=%s", GeminiAPIURL, apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	log.Println("[API-3] Sending request to Gemini...")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("[API-4] Got response with status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[ERROR] Non-200 response: %s", string(body))
		return nil, fmt.Errorf("gemini API returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	log.Printf("[API-5] Response body size: %d bytes", len(bodyBytes))
	
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response structure: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("no response from gemini")
	}

	responseText := geminiResp.Candidates[0].Content.Parts[0].Text
	
	log.Printf("[API-6] Extracted response text, length: %d chars", len(responseText))
	log.Printf("[DEBUG] Full Gemini response:\n%s\n[END OF RESPONSE]", responseText)
	
	// Write to file for debugging
	os.WriteFile("gemini_response.txt", []byte(responseText), 0644)
	log.Println("[API-6b] Wrote response to gemini_response.txt for inspection")

	// Clean response (remove markdown code blocks if present)
	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)
	
	log.Printf("[API-7] Cleaned response text, length: %d chars", len(responseText))

	// Parse JSON response
	var segments []RefinedSegment
	if err := json.Unmarshal([]byte(responseText), &segments); err != nil {
		log.Printf("[ERROR] JSON parsing failed: %v", err)
		log.Printf("[ERROR] Attempted to parse: %s", responseText)
		return nil, fmt.Errorf("failed to parse gemini response as JSON: %w", err)
	}

	// Validate segments
	if len(segments) == 0 {
		return nil, errors.New("gemini returned empty segments array")
	}
	
	log.Printf("[SUCCESS] Successfully parsed %d narration segments from Gemini", len(segments))
	for i, seg := range segments {
		log.Printf("  Segment %d: %.1f-%.1fs, %d chars", i+1, seg.Start, seg.End, len(seg.Text))
	}

	return segments, nil
}
