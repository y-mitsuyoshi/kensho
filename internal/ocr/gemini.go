package ocr

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/y-mitsuyoshi/kensho/internal/configs"
	"google.golang.org/api/option"
)

// ErrUnsupportedDocumentType is returned when the document type is not supported.
var ErrUnsupportedDocumentType = errors.New("unsupported document type")

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

	model := client.GenerativeModel("gemini-1.5-pro")
	return &Client{genaiClient: model, config: config}, nil
}

// Close closes the underlying genai client.
func (c *Client) Close(client *genai.Client) {
	if err := client.Close(); err != nil {
		log.Printf("failed to close genai client: %v", err)
	}
}

// ExtractText sends an image to the Gemini API and asks it to extract information.
func (c *Client) ExtractText(ctx context.Context, imageDatas map[string][]byte, mimeType string, docType string) (string, error) {
	doc, ok := c.config.Documents[docType]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedDocumentType, docType)
	}

	// The full prompt, including instructions and JSON structure, is now defined in the YAML config.
	prompt := []genai.Part{
		genai.Text(doc.Prompt),
	}

	// Add labeled images to the prompt in the order specified by the config
	for _, partName := range doc.ImageParts {
		imageData, ok := imageDatas[partName]
		if !ok {
			// This should have been caught by the handler, but as a safeguard:
			return "", fmt.Errorf("missing image data for part: %s", partName)
		}
		// Add a text part to label the image
		prompt = append(prompt, genai.Text(fmt.Sprintf("\nImage part: %s", partName)))
		prompt = append(prompt, genai.ImageData(mimeType, imageData))
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
