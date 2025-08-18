package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type Health struct {
	Status string `json:"status"`
}

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Health{Status: "ok"})
	})

	http.HandleFunc("/api/v1/extract", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":    "",
			"message": "stub: Gemini integration required",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
