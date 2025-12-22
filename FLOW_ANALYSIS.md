# Narration System - Complete Flow Documentation

## Overview
The system combines user speech (from Deepgram) with AI-generated narration to create a synchronized audio track for screen recordings.

## Current Architecture Flow

### Input
1. **Deepgram JSON**: User's spoken words with timestamps and confidence scores
2. **DOM Events JSON**: User interactions (clicks, scrolls, inputs) with timestamps and positions

### Processing Pipeline

#### Step 1: Normalization (`internal/normalize/time.go`)
- Filters noise events (`dom_mutation`)
- Collapses rapid scrolls (within 1.5s) into single events
- Converts absolute timestamps to video-relative times

#### Step 2: Timeline Building (`internal/timeline/build.go`)
- Merges speech words and DOM actions into single timeline
- Sorts chronologically

#### Step 3: Window Extraction (`internal/windows/extract.go`) ⚠️ **NEEDS REDESIGN**
**Current Issue**: Creates windows only in 3+ second silent gaps
**Problem**: This approach discards all user speech!

**What it should do**:
- Identify where user is actively speaking (speech-occupied zones)
- Identify true silence gaps (no speech for 2+ seconds)
- Create narration opportunities in those gaps only

#### Step 4: Script Generation (`internal/script/generate.go`)
**Current Logic**:
- For each window, check if there's speech → use speech
- Otherwise check for actions → generate description
- Apply duration constraints

**Problem**: Speech is never found because windows are created in non-speech zones

#### Step 5: Audio Generation (`internal/audio/tts.go`)
- Maps script segments to audio chunks
- Generates MP3 via Deepgram TTS
- Saves to `temp_audio/`

## Critical Issues

### Issue 1: Speech is Being Ignored
- User spoke for ~100 seconds in the video
- System only generated 1 synthetic narration
- **Root Cause**: Window extraction excludes speech-filled zones

### Issue 2: Audio Length Mismatch
- Video: 103.98 seconds
- Generated Audio: Only 1 chunk covering 38.567-103.98
- **Missing**: First 38 seconds of audio entirely

### Issue 3: No Visual Effect Sync
- Effects should trigger based on DOM events
- Should be independent of narration
- Currently coupled incorrectly

## Required Redesign

### New Approach: Hybrid Narration

```
Timeline: |--Speech--|--Silence--|--Speech--|--Action--|--Silence--|

Output:   |--User's--|--AI Gen---|--User's--|----------|--AI Gen---|
          |  Voice  |  Narration|   Voice  |          | Narration|
```

### Implementation Plan

1. **Extract Speech Zones**
   - Group consecutive speech words (with small gaps allowed)
   - These become "speech segments" that use original user voice

2. **Identify True Silence**
   - Find gaps between speech segments (2+ seconds)
   - These are opportunities for AI narration

3. **Generate Contextual Narration**
   - For each silence gap, check for DOM events
   - Generate brief descriptions if significant actions occurred

4. **Audio Assembly**
   - Option A: Use original audio file + insert TTS chunks
   - Option B: Convert entire transcript to TTS + add synthetic narration

## Questions for User

1. **Audio Source Preference**:
   - Do you have the original audio file from the recording?
   - Or should we convert the entire Deepgram transcript to TTS?

2. **Narration Style**:
   - Should we narrate every action in silence?
   - Or only significant moments (clicks, inputs)?

3. **Speech Handling**:
   - Use user's actual voice where they spoke?
   - Or convert everything to synthetic voice for consistency?
