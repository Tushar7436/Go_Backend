package main

import (
	"log"
	"net/http"

	"godemo/internal/api"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] No .env file found, using system environment variables")
	} else {
		log.Println("[INFO] Loaded environment variables from .env file")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/process-recording", api.ProcessRecording)

	// Serve generated audio files
	fs := http.FileServer(http.Dir("instructions/temp_audio"))
	mux.Handle("/audio/", http.StripPrefix("/audio/", fs))

	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,
	}

	log.Println("[INFO] Go narration engine running on :8000")
	log.Fatal(server.ListenAndServe())
}
