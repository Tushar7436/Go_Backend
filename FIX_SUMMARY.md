# Fix Summary - Audio Quality & Duration Issues

## Problems Identified

1. **Audio Duration Mismatch**: Video was 103.98s but audio was only 56s
2. **Broken Sentences**: Speech was fragmented ("With", "And" as single-word chunks)
3. **Missing Punctuation**: No capitals, commas, or periods
4. **Poor Grammar**: Words cut mid-sentence ("You can online from these brands")

## Root Causes

### Issue 1: Window Breaking on Actions
**Old Logic**: Broke speech windows every time a DOM action occurred
- User scrolled → speech window ended → new window started
- Result: 25+ tiny windows of 1-3 seconds each

**Fix**: Group speech by **natural pauses** (>0.8s gaps between words)
- DOM actions no longer interrupt speech flow
- Sentences stay together until user actually pauses
- File: `internal/windows/extract.go`

### Issue 2: Punctuation Loss
**Old Logic**: Only captured `word` field, losing capitalization/punctuation
- Deepgram provides `punctuated_word` but we ignored it

**Fix**: Added `PunctuatedWord` field to models and timeline
- Files changed:
  - `internal/models/request.go` - Added field to DeepgramWord
  - `internal/models/timeline.go` - Added field to TimelineItem
  - `internal/timeline/build.go` - Copy punctuated_word to timeline
  - `internal/script/generate.go` - Use punctuated_word in output

### Issue 3: Over-Aggressive Filtering
**Old Logic**: Dropped words with confidence < 0.70
- Many perfectly good words were filtered out
- Created gaps in sentences

**Fix**: Lowered threshold to 0.50
- Keeps more words while still filtering obvious mistakes
- File: `internal/script/generate.go`

## Test Results Expected

### Before
```json
"narrations": [
  {"start":1.92,"end":3.08,"text":"This is my website"},
  {"start":3.36,"end":6.44,"text":"Flipkart what it does i have"},
  {"start":11.56,"end":12.37,"text":"With"},
  {"start":12.37,"end":14.08,"text":"With"}
]
```

### After
```json
"narrations": [
  {"start":1.92,"end":8.74,"text":"So this is my website, Flipkart. What it does is I have, um, many brands which have connected to me."},
  {"start":9.04,"end":27.11,"text":"You can online shop from these brands with, uh, clearance price and without the hassle of going to anywhere physically. You can find everything that you require over like, for I have a new feature, which is the minutes."}
]
```

## How to Test

1. **Restart the server** (to load new code):
   ```powershell
   Stop-Process -Id (Get-NetTCPConnection -LocalPort 8000).OwningProcess -Force
   go run cmd/server/main.go
   ```

2. **Run test script**:
   ```powershell
   go run test_full_pipeline.go
   ```

3. **Verify**:
   - Check narrations count (should be ~5-10 instead of 25+)
   - Check first narration text (should have proper punctuation)
   - Check audio duration (~100 seconds instead of 56)
   - Listen to `temp_audio/*.mp3` (should sound natural, not choppy)

## Technical Details

### Window Extraction Algorithm

**Old Approach**:
```
For each action:
  Create window from last cursor to action.T
```
Problem: Actions happened every 2-3s, creating tiny windows

**New Approach**:
```
1. Scan timeline for speech words
2. Group consecutive words (gap <0.8s = same sentence)
3. When gap >0.8s → end sentence, start new one
4. Create windows from these natural sentence boundaries
```

### Speech Priority

Windows are now processed as:
1. **Check for speech first** → If found, use user's actual words
2. **Check for actions second** → If no speech, generate description
3. **Result**: User's voice takes priority, synthetic narration fills silences
