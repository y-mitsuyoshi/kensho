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

var (
	ocrClient *ocr.Client
	config    *configs.Config
)

func main() {
	ctx := context.Background()
	var err error
	config, err = configs.LoadConfig("configs/document_types.yml")
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

	// Limit request body to 100MB to avoid OOM
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)

	// We'll extract document_type from multipart parts to avoid pre-parsing the body
	var docType string

	// document type is validated by ocr client later

	// Collect all uploaded files with field name "image"
	var imageDatas [][]byte
	var mimeType string

	// Use MultipartReader to support multiple files with same field name
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
		// Handle document_type field in multipart parts
		if part.FormName() == "document_type" {
			b, _ := io.ReadAll(part)
			part.Close()
			docType = string(b)
			continue
		}
		if part.FormName() != "image" {
			// skip other non-image parts
			continue
		}
		if mimeType == "" {
			mimeType = part.Header.Get("Content-Type")
		}
		b, err := io.ReadAll(part)
		part.Close()
		if err != nil {
			http.Error(w, "Could not read file", http.StatusInternalServerError)
			return
		}
		if len(b) > 0 {
			imageDatas = append(imageDatas, b)
		}
	}

	if len(imageDatas) == 0 {
		http.Error(w, "at least one image is required", http.StatusBadRequest)
		return
	}

	if docType == "" {
		http.Error(w, "document_type is required", http.StatusBadRequest)
		return
	}

	jsonString, err := ocrClient.ExtractText(r.Context(), imageDatas, mimeType, docType)
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
