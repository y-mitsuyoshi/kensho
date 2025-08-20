# Kensho Go Library

[![Go Reference](https://pkg.go.dev/badge/github.com/y-mitsuyoshi/kensho.svg)](https://pkg.go.dev/github.com/y-mitsuyoshi/kensho)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`Kensho` is a Go library that uses Google's **Gemini 2.5 Pro** model to extract information from identity documents like driver's licenses and My Number cards with high precision, returning it as a JSON object. The name is inspired by the concept of "Kensho" (見証), which means to "see and verify."

## ✨ Features

- **High-Precision Extraction**: Leverages the Gemini 2.5 Pro model to accurately extract information even from tilted or reflective images.
- **Structured JSON Output**: Returns structured JSON, making it easy to integrate with other systems.
- **Optimized for Japanese ID Documents**: Fine-tuned for major Japanese identity documents.
- **Advanced Image Preprocessing**: Includes built-in image preprocessing features like deskewing, contrast adjustment, and noise reduction to improve OCR accuracy.
- **Simple Go Implementation**: Built with the standard library and the Google AI Go SDK for lightweight and fast performance.

## 💻 Tech Stack

- **Language**: Go
- **AI Model**: Google Gemini 2.5 Pro
- **Key Library**: [Google AI Go SDK](https://github.com/google/generative-ai-go)

## 🚀 Installation

To add Kensho to your project, use `go get`:

```bash
go get -u github.com/y-mitsuyoshi/kensho
```

##  Usage

Here is a basic example of how to use the Kensho client.

First, ensure you have set your Gemini API key as an environment variable:

```bash
export GEMINI_API_KEY="YOUR_API_KEY_HERE"
```

Then, you can use the client in your Go application:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/y-mitsuyoshi/kensho/kensho"
)

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")

	// Create a new client with the default embedded configuration
	client, err := kensho.NewClient(ctx, apiKey)
	if err != nil {
		log.Fatalf("Failed to create kensho client: %v", err)
	}
	defer client.Close()

	// Read your image file
	// In a real application, you might get this from an HTTP request or other source.
	frontImage, err := os.ReadFile("/path/to/your/image.jpg")
	if err != nil {
		log.Fatalf("Failed to read image file: %v", err)
	}

	// Prepare the file parts for the API call
	fileParts := map[string]kensho.FilePart{
		"front": {
			Content:  frontImage,
			MimeType: "image/jpeg",
		},
	}

	// Specify the document type you want to extract
	docType := "driver_license" // or "individual_number_card"

	// Call the extraction method
	data, err := client.Extract(ctx, fileParts, docType)
	if err != nil {
		log.Fatalf("Failed to extract data: %v", err)
	}

	// The result is a map[string]interface{}
	// You can easily marshal it to a JSON string for display
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println(string(prettyJSON))
}
```

## 🌐 Example: Running as a Web Service

This repository also includes a sample web server that exposes the Kensho library via an HTTP API.

### 1. Set API Key

First, copy the `.env.example` file to `.env`:

```bash
cp .env.example .env
```

Then, open `.env` and add your `GEMINI_API_KEY`.

```dotenv
# .env
PORT=8080
GEMINI_API_KEY="YOUR_API_KEY_HERE"
```

### 2. Run the Service

You can manage the service using the provided `Makefile`.

#### Start the Server

This command builds the Docker container and starts it in the background.

```bash
make up
```

#### Check Logs

```bash
make logs
```

If you see `listening on :8080`, the server is ready.

#### Send an OCR Request

From another terminal, use `curl` to send an ID document image.

- Replace `/path/to/your/image.png` with the actual file path.
- The server supports `image/png`, `image/jpeg`, and `image/webp`.
- For a driver's license (`driver_license`), you can send `image_front` and `image_back`.
- For an individual number card (`individual_number_card`), send `image_front`.

```bash
curl -X POST http://localhost:8080/api/v1/extract \
  -F "document_type=driver_license" \
  -F "image_front=@/path/to/your/image.png"
```

A successful request will return a JSON response like this:

```json
{
  "address": "東京都千代田区霞が関2-1-1",
  "birth_date": "昭和60年1月1日",
  "card_number": "第123456789012号",
  "expiry_date": "平成30年2月1日",
  "issue_date": "平成25年4月1日",
  "name": "見本太郎"
}
```

### 3. Other `make` Commands

| Command      | Description                                           |
|--------------|-------------------------------------------------------|
| `make up`    | Build and start containers in the background.         |
| `make down`  | Stop and remove containers and associated networks.   |
| `make stop`  | Stop the containers.                                  |
| `make logs`  | View the logs of the running containers.              |
| `make shell` | Start a shell inside the running `api` service container. |
| `make build` | Build the Docker image.                               |

## 📜 License

This project is released under the **MIT License**.