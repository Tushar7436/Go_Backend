package audio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"godemo/internal/models"
)

const (
	DefaultTTSSpeedWPS = 2.5   // Increased speed for more energy and excitement
	MaxTTSToleranceSec = 0.3   
)

// Supported providers
const (
	TTSDeepgram  = "deepgram"
	TTSElevenLab = "elevenlabs"
)

// SaveFullAudio generates audio for all chunks and mixes them with background music using ffmpeg.
func SaveFullAudio(chunks []models.AudioChunk, provider string, filename string, totalDuration float64) error {
	dirPath := filepath.Join("instructions", "temp_audio")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	tempFiles := []string{}
	for i, chunk := range chunks {
		audioBytes, err := GenerateAudioBytes(chunk.Text, provider)
		if err != nil {
			log.Printf("[WARN] Failed generating audio for chunk %d: %v", i, err)
			continue
		}
		tempFile := filepath.Join(dirPath, fmt.Sprintf("chunk_%d_%d.mp3", i, os.Getpid()))
		if err := os.WriteFile(tempFile, audioBytes, 0644); err != nil {
			return fmt.Errorf("failed to write temp chunk: %v", err)
		}
		tempFiles = append(tempFiles, tempFile)
	}

	if len(tempFiles) == 0 {
		return errors.New("no audio chunks generated")
	}

	fullPath := filepath.Join(dirPath, filename)
	args := []string{"-y"}
	
	// Input 0: Silent base
	args = append(args, "-f", "lavfi", "-i", fmt.Sprintf("anullsrc=r=44100:cl=stereo:d=%f", totalDuration))
	
	// Inputs 1..N: Narration chunks
	for _, f := range tempFiles {
		args = append(args, "-i", f)
	}

	// Inputs (N+1)...: Background music tracks
	musicMap := map[string]int{} // style -> input index
	styles := []string{"upbeat", "tech", "luxury", "travel", "minimal"}
	
	musicIdx := len(tempFiles) + 1
	for _, style := range styles {
		musicPath := filepath.Join("assets", "music", style+".mp3")
		if _, err := os.Stat(musicPath); err == nil {
			musicMap[style] = musicIdx
			args = append(args, "-stream_loop", "-1", "-i", musicPath)
			musicIdx++
		}
	}

	var filterParts []string
	var mixInputs []string
	mixInputs = append(mixInputs, "[0]") // silence base

	// Process Narrations (v1, v2...)
	for i, chunk := range chunks {
		if i >= len(tempFiles) { break }
		delayMs := int(chunk.Start * 1000)
		label := fmt.Sprintf("v%d", i+1)
		
		// Determine the max time this voice is allowed to talk before the next one starts
		// We add a tiny 0.2s padding so it doesn't sound like a hard cut
		durationLimit := 100.0 // default high
		if i < len(chunks)-1 && chunks[i+1].Start > chunk.Start {
			durationLimit = (chunks[i+1].Start - chunk.Start) + 0.2
		} else if i == len(chunks)-1 {
			durationLimit = (totalDuration - chunk.Start) + 0.2
		}

		// Strong broadcast voice: Volume boost + Punchy compressor + Clarity treble
		filterParts = append(filterParts, fmt.Sprintf("[%d:a]atrim=duration=%f,volume=1.5,aresample=44100,compand=0.3|0.3:1|1:-90/-60|-60/-40|-40/-30|-20/-20:6:0:-90:0.2,treble=g=5,adelay=%d|%d[%s]", i+1, durationLimit, delayMs, delayMs, label))
		mixInputs = append(mixInputs, fmt.Sprintf("[%s]", label))
	}

	// Process Background Music segments (bg1, bg2...)
	for i, chunk := range chunks {
		style := chunk.MusicStyle
		if style == "" { style = "upbeat" }
		
		inputIdx, ok := musicMap[style]
		if !ok { continue }

		label := fmt.Sprintf("bg%d", i+1)
		duration := chunk.End - chunk.Start
		if i < len(chunks)-1 && chunks[i+1].Start > chunk.Start {
			duration = chunks[i+1].Start - chunk.Start
		}
		
		if duration <= 0.1 { continue }

		delayMs := int(chunk.Start * 1000)
		// Music at 8% volume, trimmed exactly to window
		filterParts = append(filterParts, fmt.Sprintf("[%d:a]atrim=duration=%f,volume=0.08,adelay=%d|%d[%s]", inputIdx, duration, delayMs, delayMs, label))
		mixInputs = append(mixInputs, fmt.Sprintf("[%s]", label))
	}

	filterStr := strings.Join(filterParts, ";") + ";" + strings.Join(mixInputs, "") + fmt.Sprintf("amix=inputs=%d:duration=longest:dropout_transition=0:normalize=0", len(mixInputs))
	args = append(args, "-filter_complex", filterStr, "-c:a", "libmp3lame", "-q:a", "2", fullPath)

	log.Printf("[TTS] Mixing %d chunks into %s", len(tempFiles), fullPath)
	
	cmd := exec.Command("ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("[ERROR] ffmpeg failed: %s", stderr.String())
		return fmt.Errorf("ffmpeg mix error: %v", err)
	}

	for _, f := range tempFiles { os.Remove(f) }
	return nil
}

// GenerateAudioBytes calls the actual TTS provider API
func GenerateAudioBytes(text string, provider string) ([]byte, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("empty text")
	}

	// Improve punctuation for more natural pauses
	// Adding a period after an exclamation mark usually forces a better pause in Deepgram
	text = strings.ReplaceAll(text, "!", "!. ")
	text = strings.ReplaceAll(text, "?", "?. ")

	switch provider {
	case TTSDeepgram:
		apiKey := "a07edc13b7ff217582930d06b86c6487a1ae6b6f"
		// Stella is often more expressive and polished than Hera for 'encouraging' tones
		url := "https://api.deepgram.com/v1/speak?model=aura-stella-en"

		payload := map[string]string{"text": text}
		jsonData, _ := json.Marshal(payload)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Token "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("deepgram error: %s - %s", resp.Status, string(body))
		}

		return io.ReadAll(resp.Body)

	case TTSElevenLab:
		return nil, errors.New("elevenlabs not configured")
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// MapNarrationsToAudioChunks converts narration plans into time-safe audio chunk metadata.
func MapNarrationsToAudioChunks(
	narrations []models.Narration,
	windows []models.Window,
	provider string,
) ([]models.AudioChunk, error) {

	var chunks []models.AudioChunk
	for _, n := range narrations {
		if strings.TrimSpace(n.Text) == "" { continue }

		duration := n.End - n.Start
		if duration <= 0 { continue }

		// Simple mapping directly from Gemini segments
		chunk := models.AudioChunk{
			WindowIndex: n.WindowIndex,
			Start:       n.Start,
			End:         n.End,
			Text:        n.Text,
			Provider:    provider,
			MusicStyle:  n.MusicStyle,
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}
