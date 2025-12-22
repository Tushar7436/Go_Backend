# Load the JSON files
$dom = Get-Content "DOM.json" -Raw | ConvertFrom-Json
$deepgram = Get-Content "JSON.json" -Raw | ConvertFrom-Json

# Extract the words array from the Deepgram response
$words = $deepgram.results.channels[0].alternatives[0].words

# Transform DOM events to match the expected format
$transformedEvents = @()
foreach ($event in $dom.events) {
    if ($event.type -in @("click", "scroll", "input", "navigation")) {
        $transformedEvent = @{
            type = $event.type
            timestamp = $dom.startTime + $event.timestamp  # Make absolute
        }
        
        # Add target if present
        if ($event.target) {
            $transformedEvent.target = $event.target
        }
        
        # Add bounds if present
        if ($event.target.bbox) {
            $transformedEvent.bounds = @{
                x = $event.target.bbox.x
                y = $event.target.bbox.y
                width = $event.target.bbox.width
                height = $event.target.bbox.height
            }
        } else {
            # Default empty bounds for scroll events
            $transformedEvent.bounds = @{
                x = 0
                y = 0
                width = 0
                height = 0
            }
        }
        
        $transformedEvents += $transformedEvent
    }
}

# Create the combined payload
$payload = @{
    sessionId = $dom.sessionId
    recordingStartTimeMs = $dom.startTime
    videoDurationSec = $deepgram.metadata.duration
    deepgramResponse = @{
        words = $words
    }
    domEvents = $transformedEvents
}

# Convert to JSON
$jsonPayload = $payload | ConvertTo-Json -Depth 10

# Save to file
$jsonPayload | Out-File -FilePath "combined_input.json" -Encoding UTF8

Write-Host "Created combined_input.json"
Write-Host "Sending request to server..."

# Send to server
$response = Invoke-WebRequest -Uri "http://localhost:8000/process-recording" `
    -Method POST `
    -ContentType "application/json" `
    -Body $jsonPayload `
    -UseBasicParsing

Write-Host "Response Status:" $response.StatusCode
Write-Host "Response:"
$response.Content | ConvertFrom-Json | ConvertTo-Json -Depth 10

# Also save response to file
$response.Content | Out-File -FilePath "server_response.json" -Encoding UTF8
Write-Host "`nResponse saved to server_response.json"
Write-Host "Check gemini_response.txt for the raw Gemini output"
