package ocr

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/y-mitsuyoshi/kensho/internal/configs"
	"google.golang.org/api/option"
)

// ErrUnsupportedDocumentType is returned when the document type is not supported.
var ErrUnsupportedDocumentType = errors.New("unsupported document type")

// ErrUnsupportedMimeType is returned when the MIME type of a file is not supported.
var ErrUnsupportedMimeType = errors.New("unsupported MIME type")

var supportedMimeTypes = map[string]bool{
	"image/jpeg":        true,
	"image/png":         true,
	"image/webp":        true,
	"application/pdf":   true,
}

// GenerativeModel is an interface that abstracts the genai.GenerativeModel.
type GenerativeModel interface {
	GenerateContent(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error)
}

// Client holds the genai client.
type Client struct {
	genaiClient GenerativeModel
	config      *configs.Config
}

// NewClient creates a new client for the Gemini API.
func NewClient(ctx context.Context, apiKey string, config *configs.Config) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	model := client.GenerativeModel("gemini-2.5-pro")
	return &Client{genaiClient: model, config: config}, nil
}

// Close closes the underlying genai client.
func (c *Client) Close(client *genai.Client) {
	if err := client.Close(); err != nil {
		log.Printf("failed to close genai client: %v", err)
	}
}

// FilePart represents a file part with its content and MIME type.
type FilePart struct {
	Content  []byte
	MimeType string
}

// ExtractText sends one or more files (images or PDFs) to the Gemini API and asks it to extract information.
func (c *Client) ExtractText(ctx context.Context, fileParts map[string]FilePart, docType string) (string, error) {
	doc, ok := c.config.Documents[docType]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedDocumentType, docType)
	}

	prompt := []genai.Part{
		genai.Text(doc.Prompt),
	}

	for _, partName := range doc.ImageParts {
		part, ok := fileParts[partName]
		if !ok {
			return "", fmt.Errorf("missing file data for part: %s", partName)
		}

		// Clean up MIME type
		mimeType := strings.TrimSpace(part.MimeType)
		if idx := strings.Index(mimeType, ";"); idx != -1 {
			mimeType = strings.TrimSpace(mimeType[:idx])
		}

		// Validate MIME type
		if !supportedMimeTypes[mimeType] {
			// If not supported, try to detect from content
			detectedMimeType := http.DetectContentType(part.Content)
			if !supportedMimeTypes[detectedMimeType] {
				return "", fmt.Errorf("%w: %s", ErrUnsupportedMimeType, mimeType)
			}
			mimeType = detectedMimeType
		}

		// Add a text part to label the file, then the file data itself.
		prompt = append(prompt, genai.Text(fmt.Sprintf("\nFile part: %s", partName)))
		prompt = append(prompt, genai.Blob{MIMEType: mimeType, Data: part.Content})
	}

	resp, err := c.genaiClient.GenerateContent(ctx, prompt...)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	// Assuming the first part of the response is the JSON text
	if jsonText, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		raw := string(jsonText)
		cleaned := sanitizeJSONResponse(raw)
		return cleaned, nil
	}

	return "", fmt.Errorf("unexpected response format from API")
}

// sanitizeJSONResponse attempts to extract a JSON object/array from a string
// that may contain Markdown code fences or surrounding text. It returns the
// inner JSON string if found, otherwise returns the trimmed original.
func sanitizeJSONResponse(s string) string {
	s = strings.TrimSpace(s)

	// If the response contains triple-backtick fences like ```json ... ```
	// remove them by extracting the first JSON-like substring.
	// Find the first brace or bracket and the last matching closing char.
	start := strings.IndexAny(s, "{[")
	end := strings.LastIndexAny(s, "}]")
	if start != -1 && end != -1 && end >= start {
		return s[start : end+1]
	}

	// Fallback: strip backticks and trim whitespace.
	s = strings.Trim(s, "`\n \t\r")
	return s
}
