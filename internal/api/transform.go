package api

import (
	"encoding/json"
	"fmt"
	"godemo/internal/models"
)

// transformRawRequest converts the raw JSON outputs into our internal ProcessingRequest structure
func transformRawRequest(raw models.RawProcessingRequest) (*models.ProcessingRequest, error) {
	// 1. Parse Deepgram Raw
	var dg struct {
		Metadata struct {
			Duration float64 `json:"duration"`
		} `json:"metadata"`
		Results struct {
			Channels []struct {
				Alternatives []struct {
					Words []models.DeepgramWord `json:"words"`
				} `json:"alternatives"`
			} `json:"channels"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw.DeepgramRaw, &dg); err != nil {
		return nil, fmt.Errorf("failed to parse deepgramRaw: %v", err)
	}

	// 2. Parse DOM Raw
	var dr struct {
		SessionID string `json:"sessionId"`
		StartTime int64  `json:"startTime"`
		Events    []struct {
			Type      string                 `json:"type"`
			Timestamp int64                  `json:"timestamp"`
			Target    map[string]interface{} `json:"target"`
		} `json:"events"`
	}
	if err := json.Unmarshal(raw.DomRaw, &dr); err != nil {
		return nil, fmt.Errorf("failed to parse domRaw: %v", err)
	}

	// 3. Transform Events
	var transformedEvents []models.DomEvent
	for _, e := range dr.Events {
		// Only process relevant interaction events
		if e.Type == "click" || e.Type == "scroll" || e.Type == "input" || e.Type == "navigation" {
			evt := models.DomEvent{
				Type:      e.Type,
				Timestamp: dr.StartTime + e.Timestamp, // Make timestamp absolute
				Target:    e.Target,
			}

			// Extract bounds if available in target.bbox
			if e.Target != nil {
				if bbox, ok := e.Target["bbox"].(map[string]interface{}); ok {
					evt.Bounds = &models.BoundingBox{
						X:      getFloat(bbox["x"]),
						Y:      getFloat(bbox["y"]),
						Width:  getFloat(bbox["width"]),
						Height: getFloat(bbox["height"]),
					}
				}
			}
			
			// Default bounds if none found
			if evt.Bounds == nil {
				evt.Bounds = &models.BoundingBox{X: 0, Y: 0, Width: 0, Height: 0}
			}

			transformedEvents = append(transformedEvents, evt)
		}
	}

	return &models.ProcessingRequest{
		SessionID:            dr.SessionID,
		RecordingStartTimeMs: dr.StartTime,
		VideoDurationSec:     dg.Metadata.Duration,
		DeepgramResponse: &models.DeepgramResult{
			Words: dg.Results.Channels[0].Alternatives[0].Words,
		},
		DomEvents: transformedEvents,
	}, nil
}

func getFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64: return t
	case int: return float64(t)
	case int64: return float64(t)
	default: return 0
	}
}
