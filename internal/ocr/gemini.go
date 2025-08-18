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

// Client holds the genai client.
type Client struct {
	genaiClient *genai.GenerativeModel
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
func (c *Client) ExtractText(ctx context.Context, imageData []byte, mimeType string, docType string) (string, error) {
	doc, ok := c.config.Documents[docType]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedDocumentType, docType)
	}

	// Build the JSON structure part of the prompt
	var jsonFields []string
	for key, desc := range doc.JSONStructure {
		jsonFields = append(jsonFields, fmt.Sprintf(`- %s (%s)`, key, desc))
	}
	jsonStructureString := strings.Join(jsonFields, "\n")

	// Combine the prompt template with the dynamic JSON structure
	fullPrompt := fmt.Sprintf("%s\n%s", doc.Prompt, jsonStructureString)

	prompt := []genai.Part{
		genai.Text(fullPrompt),
		genai.ImageData(mimeType, imageData),
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
		return string(jsonText), nil
	}

	return "", fmt.Errorf("unexpected response format from API")
}
