# Gemini LLM Integration Setup

## Overview
The system now uses Google Gemini Flash 2.5 to refine messy speech transcripts into professional narration.

## Setup Steps

### 1. Get Gemini API Key

1. Go to [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Click "Create API Key"
3. Copy your API key

### 2. Set Environment Variable

**Windows (PowerShell)**:
```powershell
$env:GEMINI_API_KEY="YOUR_API_KEY_HERE"
```

**Windows (Permanent - System Environment Variables)**:
1. Press `Win + R`, type `sysdm.cpl`, press Enter
2. Go to "Advanced" tab → "Environment Variables"
3. Under "User variables", click "New"
4. Variable name: `GEMINI_API_KEY`
5. Variable value: Your actual API key
6. Click OK

**Linux/Mac**:
```bash
export GEMINI_API_KEY="YOUR_API_KEY_HERE"
```

### 3. Test the Integration

```powershell
# Set the API key (if not set permanently)
$env:GEMINI_API_KEY="your-actual-key-here"

# Start the server
go run cmd/server/main.go

# In a new terminal, run the test
go run test_full_pipeline.go
```

## What to Expect

### Before (Without Gemini)
- 70+ tiny fragments
- Broken sentences: "With", "And", "pretty... pretty cool"
- Grammatical errors
- Audio: ~47 seconds (missing content)

### After (With Gemini)
- 5-10 clean segments
- Professional narration: "Welcome to Flipkart, my e-commerce platform..."
- Proper grammar and punctuation
- Audio: ~100 seconds (full coverage)

## Example Output

```json
{
  "narrations": [
    {
      "windowIndex": 0,
      "start": 0,
      "end": 8.5,
      "text": "Welcome to Flipkart, my e-commerce website. I've partnered with many brands to offer online shopping at clearance prices, eliminating the need to visit physical stores."
    },
    {
      "windowIndex": 1,
      "start": 8.5,
      "end": 15.2,
      "text": "One of our newest features is Minutes delivery. It uses your location to deliver groceries, snacks, and other products in just ten minutes."
    }
  ]
}
```

## Troubleshooting

### Error: "gemini API key required"
- Make sure you set the `GEMINI_API_KEY` environment variable
- Restart your terminal after setting the variable

### Error: "gemini API returned 400"
- Check that your API key is valid
- Ensure you have API quota remaining (Gemini has free tier limits)

### Fallback Behavior
If Gemini fails or no API key is set, the system will return empty narrations.
You can add back the basic script generation as a fallback if needed.

## Cost Estimate

Gemini Flash 2.5 pricing (as of Dec 2024):
- Input: $0.075 per 1M tokens
- Output: $0.30 per 1M tokens

For a ~100 second video:
- Input tokens: ~500 (transcript + events)
- Output tokens: ~200 (refined script)
- **Cost: ~$0.0001 per video** (essentially free)

## API Limits

Free tier:
- 15 requests per minute
- 1 million tokens per day
- Should be more than enough for development/testing
