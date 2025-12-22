package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// Minimal structures to read the raw files
type RawDomJson struct {
	SessionID string `json:"sessionId"`
	StartTime int64  `json:"startTime"`
	Events    []struct {
		Type      string `json:"type"`
		Timestamp int64  `json:"timestamp"`
		Target    struct {
			Selector string `json:"selector"`
			Bbox     struct {
				X      float64 `json:"x"`
				Y      float64 `json:"y"`
				Width  float64 `json:"width"`
				Height float64 `json:"height"`
			} `json:"bbox"`
		} `json:"target"`
	} `json:"events"`
}

type RawDeepgramJson struct {
	Results struct {
		Channels []struct {
			Alternatives []struct {
				Words []struct {
					Word       string  `json:"word"`
					Start      float64 `json:"start"`
					End        float64 `json:"end"`
					Confidence float64 `json:"confidence"`
				} `json:"words"`
			} `json:"alternatives"`
		} `json:"channels"`
	} `json:"results"`
}

func main() {
	// 1. Read DOM.json
	domData, err := os.ReadFile("DOM.json")
	if err != nil {
		log.Fatal(err)
	}
	var rawDom RawDomJson
	json.Unmarshal(domData, &rawDom)

	// 2. Read JSON.json
	dgData, err := os.ReadFile("JSON.json")
	if err != nil {
		log.Fatal(err)
	}
	var rawDg RawDeepgramJson
	json.Unmarshal(dgData, &rawDg)

	// 3. Construct ProcessingRequest
	requestBody := map[string]interface{}{
		"sessionId":            rawDom.SessionID,
		"recordingStartTimeMs": rawDom.StartTime,
		"videoDurationSec":     103.98, // from JSON.json metadata duration
		"deepgramResponse": map[string]interface{}{
			"words": rawDg.Results.Channels[0].Alternatives[0].Words,
		},
		"domEvents": []map[string]interface{}{},
	}

	// Transform DOM events
	var events []map[string]interface{}
	for _, e := range rawDom.Events {
		if e.Type == "click" || e.Type == "input" || e.Type == "scroll" {
			event := map[string]interface{}{
				"type":      e.Type,
				"timestamp": rawDom.StartTime + e.Timestamp, // make absolute
				"target": map[string]interface{}{
					"selector": e.Target.Selector,
				},
				"bounds": map[string]interface{}{
					"x":      e.Target.Bbox.X,
					"y":      e.Target.Bbox.Y,
					"width":  e.Target.Bbox.Width,
					"height": e.Target.Bbox.Height,
				},
			}
			events = append(events, event)
		}
	}
	requestBody["domEvents"] = events

	// 4. Send to Server
	payload, _ := json.MarshalIndent(requestBody, "", "  ")
	resp, err := http.Post("http://localhost:8000/process-recording", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Error: Server might not be running. Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 5. Read Result
	result, _ := io.ReadAll(resp.Body)
	fmt.Println("--- RESPONSE FROM SERVER ---")
	fmt.Println(string(result))
	fmt.Println("----------------------------")
	fmt.Println("Check the 'internal/instructions/temp_audio' folder for the .mp3 file.")
}
