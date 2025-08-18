package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/y-mitsuyoshi/kensho/internal/configs"
	"github.com/y-mitsuyoshi/kensho/internal/ocr"
)

type Health struct {
	Status string `json:"status"`
}

// Global OCR client
var ocrClient *ocr.Client

func main() {
	ctx := context.Background()
	config, err := configs.LoadConfig("configs/document_types.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ocrClient, err = ocr.NewClient(ctx, os.Getenv("GEMINI_API_KEY"), config)
	if err != nil {
		log.Fatalf("Failed to create OCR client: %v", err)
	}
	// Note: The genai.Client used to create the model is not directly exposed,
	// so we can't defer its closure here. We'll rely on the application lifecycle
	// to manage the connection.

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/v1/extract", extractHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Health{Status: "ok"})
}

func extractHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Max file size: 10MB
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Could not parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Could not get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Could not read file", http.StatusInternalServerError)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	docType := r.FormValue("document_type")
	if docType == "" {
		http.Error(w, "document_type is required", http.StatusBadRequest)
		return
	}

	jsonString, err := ocrClient.ExtractText(r.Context(), imgBytes, mimeType, docType)
	if err != nil {
		if errors.Is(err, ocr.ErrUnsupportedDocumentType) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("Error from OCR client: %v", err)
		http.Error(w, fmt.Sprintf("Failed to extract text: %v", err), http.StatusInternalServerError)
		return
	}

	// Validate that the string from Gemini is valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonString), &data); err != nil {
		log.Printf("Error unmarshalling JSON from Gemini: %v. Raw response: %s", err, jsonString)
		http.Error(w, "Failed to parse OCR response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}
