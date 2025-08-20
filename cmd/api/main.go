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
	"strings"

	"github.com/y-mitsuyoshi/kensho/kensho"
)

type Health struct {
	Status string `json:"status"`
}

var (
	kenshoClient *kensho.Client
)

func main() {
	ctx := context.Background()
	var err error
	configPath := "configs/document_types.yml"

	kenshoClient, err = kensho.NewClient(ctx, os.Getenv("GEMINI_API_KEY"), configPath)
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

	// Limit request body to 100MB to avoid OOM
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)

	var docType string
	fileParts := make(map[string]kensho.FilePart)

	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "Could not parse multipart data", http.StatusBadRequest)
		return
	}

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "Error reading multipart data", http.StatusBadRequest)
			return
		}

		formName := part.FormName()
		if formName == "document_type" {
			b, err := io.ReadAll(part)
			part.Close()
			if err != nil {
				http.Error(w, "Could not read document_type", http.StatusInternalServerError)
				return
			}
			docType = string(b)
			continue
		}

		if strings.HasPrefix(formName, "image_") {
			mimeType := part.Header.Get("Content-Type")
			b, err := io.ReadAll(part)
			part.Close()
			if err != nil {
				http.Error(w, "Could not read file", http.StatusInternalServerError)
				return
			}
			if len(b) > 0 {
				key := strings.TrimPrefix(formName, "image_")
				fileParts[key] = kensho.FilePart{Content: b, MimeType: mimeType}
			}
			continue
		}

		// Close part if it's not used
		part.Close()
	}

	if docType == "" {
		http.Error(w, "document_type is required", http.StatusBadRequest)
		return
	}

	if len(fileParts) == 0 {
		http.Error(w, "at least one image is required", http.StatusBadRequest)
		return
	}

	jsonString, err := kenshoClient.ExtractText(r.Context(), fileParts, docType)
	if err != nil {
		if errors.Is(err, kensho.ErrUnsupportedDocumentType) || errors.Is(err, kensho.ErrUnsupportedMimeType) {
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
