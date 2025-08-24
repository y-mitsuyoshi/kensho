package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

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

	// The client now uses the default embedded configuration.
	modelName := os.Getenv("GEMINI_MODEL") // Read the model name from environment variable
	kenshoClient, err = kensho.NewClient(ctx, os.Getenv("GEMINI_API_KEY"), modelName)
	if err != nil {
		log.Fatalf("Failed to create kensho client: %v", err)
	}
	// Defer closing the client to clean up resources.
	defer kenshoClient.Close()

	// Serve static files from the "static" directory
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

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
	docType, fileParts, masking, preprocess, err := kensho.ParseRequest(r)
	if err != nil {
		switch {
		case errors.Is(err, kensho.ErrRequestBodyTooLarge):
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		case errors.Is(err, kensho.ErrMissingField):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, fmt.Sprintf("Could not parse request: %v", err), http.StatusBadRequest)
		}
		return
	}

	result, err := kenshoClient.Extract(r.Context(), fileParts, docType, masking, preprocess)
	if err != nil {
		if errors.Is(err, kensho.ErrUnsupportedDocumentType) || errors.Is(err, kensho.ErrUnsupportedMimeType) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("Error from kensho client: %v", err)
		http.Error(w, fmt.Sprintf("Failed to extract data: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
